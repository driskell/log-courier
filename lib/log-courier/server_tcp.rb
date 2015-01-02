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
    attr_reader :peer

    def initialise
      reset_peer
      super
    end

    # Save the peer
    def accept
      sock = super
      peer = sock.peeraddr(:numeric)
      @peer = "#{peer[2]}:#{peer[1]}"
      return sock
    end

    def reset_peer
      @peer = 'unknown'
      return
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
        ssl_verify_ca:         nil,
        max_packet_size:       10_485_760,
        add_peer_fields:       false,
      }.merge!(options)

      @logger = @options[:logger]

      if @options[:transport] == 'tls'
        [:ssl_certificate, :ssl_key].each do |k|
          fail "input/courier: '#{k}' is required" if @options[k].nil?
        end

        if @options[:ssl_verify] and (!@options[:ssl_verify_default_ca] && @options[:ssl_verify_ca].nil?)
          fail 'input/courier: Either \'ssl_verify_default_ca\' or \'ssl_verify_ca\' must be specified when ssl_verify is true'
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

          # Create the OpenSSL server - set start_immediately to false so we can multithread handshake
          @server = OpenSSL::SSL::SSLServer.new(@tcp_server, ssl)
          @server.start_immediately = false
        else
          @server = @tcp_server
        end

        if @options[:port] == 0
          @logger.warn 'Ephemeral port allocated', :transport => @options[:transport], :port => @port unless @logger.nil?
        end
      rescue => e
        raise "input/courier: Failed to initialise: #{e}"
      end
    end # def initialize

    def run(&block)
      client_threads = {}

      loop do
        # Because start_immediately is false, TCP accept is single thread but
        # handshake is essentiall multithreaded as we defer it to the thread
        @tcp_server.reset_peer
        client = nil
        begin
          client = @server.accept
        rescue EOFError, OpenSSL::SSL::SSLError, IOError => e
          # Accept failure or other issue
          @logger.warn 'Connection failed to accept', :error => e.message, :peer => @tcp_server.peer unless @logger.nil
          client.close rescue nil unless client.nil?
          next
        end

    	  @logger.info 'New connection', :peer => @tcp_server.peer unless @logger.nil?

        # Clear up finished threads
        client_threads.delete_if do |_, thr|
          !thr.alive?
        end

        # Start a new connection thread
        client_threads[client] = Thread.new(client, @tcp_server.peer) do |client_copy, peer_copy|
          run_thread client_copy, peer_copy, &block
        end
      end
      return
    rescue ShutdownSignal
      return
    rescue StandardError, NativeException => e
      # Some other unknown problem
      @logger.warn e, :hint => 'Unknown error, shutting down' unless @logger.nil?
      return
    ensure
      # Raise shutdown in all client threads and join then
      client_threads.each do |_, thr|
        thr.raise ShutdownSignal
      end

      client_threads.each(&:join)

      @tcp_server.close
    end

    private

    def run_thread(client, peer, &block)
      # Perform the handshake inside the new thread so we don't block TCP accept
      if @options[:transport] == 'tls'
        begin
          client.accept
        rescue EOFError, OpenSSL::SSL::SSLError, IOError => e
          # Handshake failure or other issue
          @logger.warn 'Connection failed to initialise', :error => e.message, :peer => peer unless @logger.nil?
          client.close
          return
        end
      end

      ConnectionTcp.new(@logger, client, peer, @options).run(&block)
    end
  end

  # Representation of a single connected client
  class ConnectionTcp
    attr_accessor :peer

    def initialize(logger, fd, peer, options)
      @logger = logger
      @fd = fd
      @peer = peer
      @peer_fields = {}
      @in_progress = false
      @options = options

      if @options[:add_peer_fields]
        @peer_fields['peer'] = peer
        if @options[:transport] == 'tls' && !@fd.peer_cert.nil?
          @peer_fields['peer_ssl_cn'] = get_cn(@fd.peer_cert)
        end
      end
    end

    def add_fields(event)
      event.merge! @peer_fields if @peer_fields.length != 0
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
        if length > @options[:max_packet_size]
          fail ProtocolError, "packet too large (#{length} > #{@options[:max_packet_size]})"
        end

        # While we're processing, EOF is bad as it may occur during send
        @in_progress = true

        # Read the message
        if length == 0
          data = ''
        else
          data = recv(length)
        end

        # Send for processing
        yield signature, data, self

        # If we EOF next it's a graceful close
        @in_progress = false
      end
      return
    rescue TimeoutError
      # Timeout of the connection, we were idle too long without a ping/pong
      @logger.warn 'Connection timed out', :peer => @peer unless @logger.nil?
      return
    rescue EOFError
      if @in_progress
        @logger.warn 'Unexpected EOF', :peer => @peer unless @logger.nil?
      else
        @logger.info 'Connection closed', :peer => @peer unless @logger.nil?
      end
      return
    rescue OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
      # Read errors, only action is to shutdown which we'll do in ensure
      @logger.warn 'SSL error, connection aborted', :error => e.message, :peer => @peer unless @logger.nil?
      return
    rescue ProtocolError => e
      # Connection abort request due to a protocol error
      @logger.warn 'Protocol error, connection aborted', :error => e.message, :peer => @peer unless @logger.nil?
      return
    rescue ShutdownSignal
      # Shutting down
      @logger.info 'Server shutting down, closing connection', :peer => @peer unless @logger.nil?
      return
    rescue StandardError, NativeException => e
      # Some other unknown problem
      @logger.warn e, :hint => 'Unknown error, connection aborted', :peer => @peer unless @logger.nil?
      return
    ensure
      @fd.close rescue nil
    end

    def send(signature, message)
      reset_timeout
      data = signature + [message.length].pack('N') + message
      done = 0
      loop do
        begin
          written = @fd.write_nonblock(data[done...data.length])
        rescue IO::WaitReadable
          fail TimeoutError if IO.select([@fd], nil, [@fd], @timeout - Time.now.to_i).nil?
          retry
        rescue IO::WaitWritable
          fail TimeoutError if IO.select(nil, [@fd], [@fd], @timeout - Time.now.to_i).nil?
          retry
        end
        fail ProtocolError, "write failure (#{done}/#{data.length})" if written == 0
        done += written
        break if done >= data.length
      end
      return
    end

    private

    def get_cn(cert)
      cert.subject.to_a.find do |oid, value|
        return value if oid == "CN"
      end
      nil
    end

    def recv(need)
      reset_timeout
      have = ''
      loop do
        begin
       	  buffer = @fd.read_nonblock need - have.length
        rescue IO::WaitReadable
          fail TimeoutError if IO.select([@fd], nil, [@fd], @timeout - Time.now.to_i).nil?
          retry
        rescue IO::WaitWritable
          fail TimeoutError if IO.select(nil, [@fd], [@fd], @timeout - Time.now.to_i).nil?
          retry
        end
        if buffer.nil?
          fail EOFError
        elsif buffer.length == 0
          fail ProtocolError, "read failure (#{have.length}/#{need})"
        end
        if have.length == 0
          have = buffer
        else
          have << buffer
        end
        break if have.length >= need
      end
      have
    end

    def reset_timeout
      # TODO: Make configurable
      @timeout = Time.now.to_i + 1_800
      return
    end
  end
end
