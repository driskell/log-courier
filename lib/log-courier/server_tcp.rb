# encoding: utf-8

# Copyright 2014 Jason Woods.
#
# This file is a modification of code from Logstash Forwarder.
# Copyright 2012-2013 Jordan Sissel and contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

require 'openssl'
require 'socket'
require 'thread'

module LogCourier
  # Wrap around TCPServer to grab last error for use in reporting which peer had an error
  class ExtendedTCPServer < TCPServer
    # Yield the peer
    def accept
      sock = super
      peer = sock.peeraddr(:numeric)
      Thread.current['LogCourierPeer'] = "#{peer[2]}:#{peer[1]}"
      return sock
    end
  end

  # TLS transport implementation for server
  class ServerTcp
    attr_reader :port

    # Create a new TLS transport endpoint
    def initialize(options = {})
      @options = {
        logger:                nil,
        transport:             'tls',
        port:                  0,
        address:               '0.0.0.0',
        ssl_certificate:       nil,
        ssl_key:               nil,
        ssl_key_passphrase:    nil,
        ssl_verify:            false,
        ssl_verify_default_ca: false,
        ssl_verify_ca:         nil
      }.merge!(options)

      @logger = @options[:logger]

      if @options[:transport] == 'tls'
        [:ssl_certificate, :ssl_key].each do |k|
          raise "[LogCourierServer] '#{k}' is required" if @options[k].nil?
        end

        if @options[:ssl_verify] and (not @options[:ssl_verify_default_ca] && @options[:ssl_verify_ca].nil?)
          raise '[LogCourierServer] Either \'ssl_verify_default_ca\' or \'ssl_verify_ca\' must be specified when ssl_verify is true'
        end
      end

      begin
        @tcp_server = ExtendedTCPServer.new(@options[:address], @options[:port])

        # Query the port in case the port number is '0'
        # TCPServer#addr == [ address_family, port, address, address ]
        @port = @tcp_server.addr[1]

        if @options[:transport] == 'tls'
          ssl = OpenSSL::SSL::SSLContext.new
          ssl.cert = OpenSSL::X509::Certificate.new(File.read(@options[:ssl_certificate]))
          ssl.key = OpenSSL::PKey::RSA.new(File.read(@options[:ssl_key]), @options[:ssl_key_passphrase])

          if @options[:ssl_verify]
            cert_store = OpenSSL::X509::Store.new

            # Load the system default certificate path to the store
            cert_store.set_default_paths if @options[:ssl_verify_default_ca]

            if File.directory?(@options[:ssl_verify_ca])
              cert_store.add_path(@options[:ssl_verify_ca])
            else
              cert_store.add_file(@options[:ssl_verify_ca])
            end

            ssl.cert_store = cert_store

            ssl.verify_mode = OpenSSL::SSL::VERIFY_PEER | OpenSSL::SSL::VERIFY_FAIL_IF_NO_PEER_CERT
          end

          @server = OpenSSL::SSL::SSLServer.new(@tcp_server, ssl)
        else
          @server = @tcp_server
        end

        if @options[:port] == 0
          @logger.warn '[LogCourierServer] Transport is listening on ephemeral port ' + @port.to_s
        end
      rescue => e
        raise "[LogCourierServer] Failed to initialise: #{e}"
      end
    end # def initialize

    def run(&block)
      client_threads = {}

      loop do
        # This means ssl accepting is single-threaded.
        begin
          client = @server.accept
        rescue EOFError, OpenSSL::SSL::SSLError, IOError => e
          # Handshake failure or other issue
          peer = Thread.current['LogCourierPeer'] || 'unknown'
          @logger.warn "[LogCourierServer] Connection from #{peer} failed to initialise: #{e}" unless @logger.nil?
          client.close rescue nil
          next
        end

        peer = Thread.current['LogCourierPeer'] || 'unknown'

    	  @logger.info "[LogCourierServer] New connection from #{peer}" unless @logger.nil?

        # Clear up finished threads
        client_threads.delete_if do |_, thr|
          !thr.alive?
        end

        # Start a new connection thread
        client_threads[client] = Thread.new(client, peer) do |client_copy, peer_copy|
          ConnectionTcp.new(@logger, client_copy, peer_copy).run(&block)
        end
      end
    rescue ShutdownSignal
      # Capture shutting down signal
      0
    ensure
      # Raise shutdown in all client threads and join then
      client_threads.each do |_, thr|
        thr.raise ShutdownSignal
      end

      client_threads.each(&:join)

      @tcp_server.close
    end
  end

  # Representation of a single connected client
  class ConnectionTcp
    attr_accessor :peer

    def initialize(logger, fd, peer)
      @logger = logger
      @fd = fd
      @peer = peer
      @in_progress = false
    end

    def run
      loop do
        # Read messages
        # Each message begins with a header
        # 4 byte signature
        # 4 byte length
        # Normally we would not parse this inside transport, but for TLS we have to in order to locate frame boundaries
        signature, length = recv(8).unpack('A4N')

        # Sanity
        if length > 1_048_576
          # TODO: log something
          raise ProtocolError
        end

        # While we're processing, EOF is bad as it may occur during send
        @in_progress = true

        # Read the message
        yield signature, recv(length), self

        # If we EOF next it's a graceful close
        @in_progress = false
      end
    rescue TimeoutError
      # Timeout of the connection, we were idle too long without a ping/pong
      @logger.warn("[LogCourierServer] Connection from #{@peer} timed out") unless @logger.nil?
    rescue EOFError
      if @in_progress
        @logger.warn("[LogCourierServer] Premature connection close on connection from #{@peer}") unless @logger.nil?
      else
        @logger.info("[LogCourierServer] Connection from #{@peer} closed") unless @logger.nil?
      end
    rescue OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
      # Read errors, only action is to shutdown which we'll do in ensure
      @logger.warn("[LogCourierServer] SSL error on connection from #{@peer}: #{e}") unless @logger.nil?
    rescue ProtocolError => e
      # Connection abort request due to a protocol error
      @logger.warn("[LogCourierServer] Protocol error on connection from #{@peer}: #{e}") unless @logger.nil?
    rescue ShutdownSignal
      # Shutting down
      @logger.warn("[LogCourierServer] Closing connecting from #{@peer}: server shutting down") unless @logger.nil?
    rescue => e
      # Some other unknown problem
      @logger.warn("[LogCourierServer] Unknown error on connection from #{@peer}: #{e}") unless @logger.nil?
      @logger.debug("[LogCourierServer] #{e.backtrace}: #{e.message} (#{e.class})") unless @logger.nil? || !@logger.debug?
    ensure
      @fd.close rescue nil
    end

    def recv(need)
      reset_timeout
      begin
        buffer = @fd.read_nonblock need
      rescue IO::WaitReadable
        raise TimeoutError if IO.select([@fd], nil, [@fd], @timeout - Time.now.to_i).nil?
        retry
      rescue IO::WaitWritable
        raise TimeoutError if IO.select(nil, [@fd], [@fd], @timeout - Time.now.to_i).nil?
        retry
      end
      if buffer.nil?
        raise EOFError
      elsif buffer.length < need
        raise ProtocolError
      end
      buffer
    end

    def send(signature, message)
      reset_timeout

      written = 0
      data = signature + [message.length].pack('N') + message
      begin
        written = @fd.write_nonblock(data[written...data.length])
      rescue IO::WaitReadable
        raise TimeoutError if IO.select([@fd], nil, [@fd], @timeout - Time.now.to_i).nil?
        retry
      rescue IO::WaitWritable
        raise TimeoutError if IO.select(nil, [@fd], [@fd], @timeout - Time.now.to_i).nil?
        retry
      end
    end

    def reset_timeout
      # TODO: Make configurable
      @timeout = Time.now.to_i + 1_800
    end
  end
end
