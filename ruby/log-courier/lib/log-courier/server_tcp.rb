# Copyright 2014-2021 Jason Woods and Contributors.
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
      # Prevent reverse lookup by passing false
      begin
        peer = sock.peeraddr(false)
      rescue ArgumentError
        # Logstash <= 1.5.0 has a patch that blocks parameters (elastic/logstash#3364)
        peer = sock.peeraddr
      end
      @peer = "#{peer[2]}:#{peer[1]}"
      sock
    end

    def reset_peer
      @peer = 'unknown'
      nil
    end
  end

  # TLS transport implementation for server
  class ServerTcp
    attr_reader :port

    # Create a new TLS transport endpoint
    def initialize(options = {})
      @options = {
        logger: nil,
        transport: 'tls',
        port: 0,
        address: '0.0.0.0',
        ssl_certificate: nil,
        ssl_key: nil,
        ssl_key_passphrase: nil,
        ssl_verify: false,
        ssl_verify_default_ca: false,
        ssl_verify_ca: nil,
        max_packet_size: 10_485_760,
        add_peer_fields: false,
        min_tls_version: 1.2,
        disable_handshake: false,
      }.merge!(options)

      @logger = @options[:logger]

      if @options[:transport] == 'tls'
        [:ssl_certificate, :ssl_key].each do |k|
          raise "input/courier: '#{k}' is required" if @options[k].nil?
        end

        if @options[:ssl_verify] && (!@options[:ssl_verify_default_ca] && @options[:ssl_verify_ca].nil?)
          raise 'input/courier: Either \'ssl_verify_default_ca\' or \'ssl_verify_ca\' must be specified when ssl_verify is true'
        end
      end

      begin
        @tcp_server = ExtendedTCPServer.new(@options[:address], @options[:port])

        # Query the port in case the port number is '0'
        # TCPServer#addr == [ address_family, port, address, address ]
        @port = @tcp_server.addr[1]

        if @options[:transport] == 'tls'
          ssl = OpenSSL::SSL::SSLContext.new

          # Disable SSLv2 and SSLv3
          # Call set_params first to ensure options attribute is there (hmmmm?)
          ssl.set_params
          # Modify the default options to ensure SSLv2 and SSLv3 is disabled
          # This retains any beneficial options set by default in the current Ruby implementation
          # TODO: https://github.com/jruby/jruby-openssl/pull/215 is fixed in JRuby 9.3.0.0
          #       As of 7.15 Logstash, JRuby version is still 9.2
          #       Once 9.3 is in use we can switch to using min_version and max_version
          ssl.options |= OpenSSL::SSL::OP_NO_SSLv2
          ssl.options |= OpenSSL::SSL::OP_NO_SSLv3
          ssl.options |= OpenSSL::SSL::OP_NO_TLSv1 if @options[:min_tls_version] > 1
          ssl.options |= OpenSSL::SSL::OP_NO_TLSv1_1 if @options[:min_tls_version] > 1.1
          ssl.options |= OpenSSL::SSL::OP_NO_TLSv1_2 if @options[:min_tls_version] > 1.2
          raise 'Invalid min_tls_version - max is 1.3' if @options[:min_tls_version] > 1.3

          # Set the certificate file
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

        @logger&.warn 'Ephemeral port allocated', transport: @options[:transport], port: @port if @options[:port].zero?
      rescue StandardError => e
        raise "input/courier: Failed to initialise: #{e}"
      end
    end

    def run(&block)
      client_threads = {}

      loop do
        # Because start_immediately is false, TCP accept is single thread but
        # handshake is essentiall multithreaded as we defer it to the thread
        @tcp_server.reset_peer
        client = nil
        begin
          client = @server.accept
        rescue OpenSSL::SSL::SSLError, IOError => e
          # Accept failure or other issue
          @logger&.warn 'Connection failed to accept', error: e.message, peer: @tcp_server.peer
          begin
            client&.close
          rescue OpenSSL::SSL::SSLError, IOError
            # Ignore IO error during close
          end
          next
        end

        @logger&.info 'New connection', peer: @tcp_server.peer

        # Clear up finished threads
        client_threads.delete_if do |_, thr|
          !thr.alive?
        end

        # Start a new connection thread
        client_threads[client] = Thread.new(client, @tcp_server.peer) do |client_copy, peer_copy|
          run_thread client_copy, peer_copy, &block
        end
      end
      nil
    rescue ShutdownSignal
      nil
    rescue StandardError => e
      # Some other unknown problem
      @logger&.warn e.message, hint: 'Unknown error, shutting down'
      nil
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
        rescue OpenSSL::SSL::SSLError, IOError => e
          # Handshake failure or other issue
          @logger&.warn 'Connection failed to initialise', error: e.message, peer: peer
          begin
            client.close
          rescue OpenSSL::SSL::SSLError, IOError
            # Ignore during close
          end
          return
        end

        @logger&.info 'Connection setup successfully', peer: peer, ssl_version: client.ssl_version
      end

      ConnectionTcp.new(@logger, client, peer, @options).run(&block)
    rescue ShutdownSignal
      # Shutting down
      @logger&.info 'Server shutting down, connection closed', peer: peer
    end
  end

  # Representation of a single connected client
  class ConnectionTcp
    attr_accessor :peer

    def initialize(logger, sfd, peer, options)
      @logger = logger
      @fd = sfd
      @peer = peer
      @peer_fields = {}
      @in_progress = false
      @options = options
      @client = 'Unknown'
      @major_version = 0
      @minor_version = 0
      @patch_version = 0
      @version = '0.0.0'
      @client_version = 'Unknown'

      return unless @options[:add_peer_fields]

      @peer_fields['peer'] = peer
      return unless @options[:transport] == 'tls' && !@fd.peer_cert.nil?

      @peer_fields['peer_ssl_cn'] = get_cn(@fd.peer_cert)
    end

    def add_fields(event)
      event.merge! @peer_fields unless @peer_fields.empty?
    end

    def run(&block)
      handshake(&block)

      loop do
        signature, data = receive

        # Send for processing
        yield signature, data, self
      end
    rescue TimeoutError
      # Timeout of the connection, we were idle too long without a ping/pong
      @logger&.warn 'Connection timed out', peer: @peer
      nil
    rescue EOFError
      if @in_progress
        @logger&.warn 'Unexpected EOF', peer: @peer
      else
        @logger&.info 'Connection closed', peer: @peer
      end
      nil
    rescue OpenSSL::SSL::SSLError => e
      # Read errors, only action is to shutdown which we'll do in ensure
      @logger&.warn 'SSL error, connection aborted', error: e.message, peer: @peer
      nil
    rescue IOError, SystemCallError => e
      # Read errors, only action is to shutdown which we'll do in ensure
      @logger&.warn 'Connection aborted', error: e.message, peer: @peer
      nil
    rescue ProtocolError => e
      # Connection abort request due to a protocol error
      @logger&.warn 'Protocol error, connection aborted', error: e.message, peer: @peer
      nil
    rescue ShutdownSignal
      # Shutting down
      @logger&.info 'Server shutting down, closing connection', peer: @peer
      nil
    rescue StandardError => e
      # Some other unknown problem
      @logger&.warn e.message, hint: 'Unknown error, connection aborted', peer: @peer
      nil
    ensure
      begin
        @fd.close
      rescue OpenSSL::SSL::SSLError, IOError
        # Ignore during close
      end
    end

    def receive
      # Read message
      # Each message begins with a header
      # 4 byte signature
      # 4 byte length
      # Normally we would not parse this inside transport, but for TLS we have to in order to locate frame boundaries
      signature, length = recv(8).unpack('A4N')

      # Sanity
      raise ProtocolError, "packet too large (#{length} > #{@options[:max_packet_size]})" if length > @options[:max_packet_size]

      # While we're processing, EOF is bad as it may occur during send
      @in_progress = true

      # Read the message
      data = if length.zero?
               ''
             else
               recv(length)
             end

      # If we EOF next it's a graceful close
      @in_progress = false

      [signature, data]
    end

    def send(signature, message)
      reset_timeout
      data = signature + [message.length].pack('N') + message
      done = 0
      loop do
        begin
          written = @fd.write_nonblock(data[done...data.length])
        rescue IO::WaitReadable
          raise TimeoutError if IO.select([@fd], nil, [@fd], @timeout - Time.now.to_i).nil?

          retry
        rescue IO::WaitWritable
          raise TimeoutError if IO.select(nil, [@fd], [@fd], @timeout - Time.now.to_i).nil?

          retry
        end
        raise ProtocolError, "write failure (#{done}/#{data.length})" if written.zero?

        done += written
        break if done >= data.length
      end
      nil
    end

    private

    def handshake
      return if @options[:disable_handshake]

      signature, data = receive
      if signature == 'JDAT'
        @helo = Protocol.parse_helo_vers('')
        @logger&.info 'Remote does not support protocol handshake', peer: @peer
        yield signature, data, self
        return
      elsif signature != 'HELO'
        raise ProtocolError, "unexpected #{signature} message"
      end

      @helo = Protocol.parse_helo_vers(data)
      @logger&.info 'Remote identified', peer: @peer, client_version: @helo[:client_version]

      # Flags 4 bytes - EVNT flag = 0
      # (Significant rewrite would be required to support streaming messages as currently we read
      #  first and then yield for processing. To support EVNT we have to move protocol parsing to
      #  the connection layer here so we can keep reading until we reach the end of the stream)
      # Major Version 4 bytes
      # Minor Version 4 bytes
      # Patch Version 4 bytes
      # Client String 4 bytes
      data = [0, MAJOR_VERSION, MINOR_VERSION, PATCH_VERSION, 'RYLC'].pack('NNNNA4')
      send 'VERS', data
    end

    def get_cn(cert)
      cert.subject.to_a.find do |oid, value|
        return value if oid == 'CN'
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
          raise TimeoutError if IO.select([@fd], nil, [@fd], @timeout - Time.now.to_i).nil?

          retry
        rescue IO::WaitWritable
          raise TimeoutError if IO.select(nil, [@fd], [@fd], @timeout - Time.now.to_i).nil?

          retry
        end
        raise EOFError if buffer.nil?
        raise ProtocolError, "read failure (#{have.length}/#{need})" if buffer.length.zero?

        if have.length.zero?
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
      nil
    end
  end
end
