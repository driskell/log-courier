require "socket"
require "thread"
require "timeout"
require "openssl"
require "zlib"

module Lumberjack2
  # Wrap around TCPServer to grab last error for use in reporting which peer had an error
  class ExtendedTCPServer < TCPServer
    # Yield the peer
    def accept
      sock = super
      peer = sock.peeraddr(:numeric)
      Thread.current["LumberjackPeer"] = "#{peer[2]}:#{peer[1]}"
      return sock
    end
  end

  class ServerTls
    attr_reader :port

    # Create a new TLS transport endpoint
    def initialize(options={})
      @options = {
        :port => 0,
        :address => "0.0.0.0",
        :ssl_certificate => nil,
        :ssl_key => nil,
        :ssl_key_passphrase => nil,
        :ssl_verify => false,
        :ssl_verify_default_ca => false,
        :ssl_verify_ca => nil,
        :logger => nil,
      }.merge(options)

      @logger = @options[:logger]

      [:ssl_certificate, :ssl_key].each do |k|
        if @options[k].nil?
          raise "You must specify #{k} in Lumberjack::Server.new(...)"
        end
      end

      if @options[:ssl_verify] and (not @options[:ssl_verify_default_ca] and @options[:ssl_verify_ca].nil?)
        raise "You must specify one of ssl_verify_default_ca and ssl_verify_ca in Lumberjack::Server.new(...) when ssl_verify is true"
      end

      @tcp_server = ExtendedTCPServer.new(@options[:port])

      # Query the port in case the port number is '0'
      # TCPServer#addr == [ address_family, port, address, address ]
      @port = @tcp_server.addr[1]

      @ssl = OpenSSL::SSL::SSLContext.new
      @ssl.cert = OpenSSL::X509::Certificate.new(File.read(@options[:ssl_certificate]))
      @ssl.key = OpenSSL::PKey::RSA.new(File.read(@options[:ssl_key]), @options[:ssl_key_passphrase])

      if @options[:ssl_verify]
        @cert_store = OpenSSL::X509::Store.new

        if @options[:ssl_verify_default_ca]
          # Load the system default certificate path to the store
          @cert_store.set_default_paths
        end

        if File.directory?(@options[:ssl_verify_ca])
          @cert_store.add_path(@options[:ssl_verify_ca])
        else
          @cert_store.add_file(@options[:ssl_verify_ca])
        end

        @ssl.cert_store = @cert_store

        @ssl.verify_mode = OpenSSL::SSL::VERIFY_PEER|OpenSSL::SSL::VERIFY_FAIL_IF_NO_PEER_CERT
      end

      @ssl_server = OpenSSL::SSL::SSLServer.new(@tcp_server, @ssl)
    end # def initialize

    def run(&block)
      client_threads = Hash.new
      @logger.warn("ServerTls running")
      while true
        # This means ssl accepting is single-threaded.
        begin
          client = @ssl_server.accept
        rescue EOFError, OpenSSL::SSL::SSLError, IOError => e
          # Handshake failure or other issue
          peer = Thread.current["LumberjackPeer"] || "unknown"
          @logger.warn "[LumberjackTLS] Connection from #{peer} failed to initialise: #{e}" if not @logger.nil?
          client.close rescue nil
          next
        end

        peer = Thread.current["LumberjackPeer"] || "unknown"

    	  @logger.info "[LumberjackTLS] New connection from #{peer}" if not @logger.nil?

        # Clear up finished threads
        client_threads.delete_if do |k, thr|
          not thr.alive?
        end

        # Start a new connection thread
        client_threads[client] = Thread.new(client, peer) do |client, peer|
          @logger.warn("ServerTls new client from #{peer}")
          ConnectionTls.new(@logger, client, peer).run &block
        end

    	  # Reset client so if ssl_server.accept fails, we don't close the previous connection within rescue
    	  client = nil
      end
    rescue => e
      @logger.warn("ServerTls error: #{e}")
    ensure
      # Raise shutdown in all client threads and join then
      client_threads.each do |thr|
        thr.raise ShutdownSignal
      end

      client_threads.each &:join

      @logger.warn("ServerTls shutdown")
    end # ensure
  end # class ServerTls

  class ConnectionTls
    def initialize(logger, fd, peer)
      @logger = logger
      @fd = fd
      @peer = peer
    end

    def run
      @logger.warn("ServerTls #{@peer} running")
      while true
        # Read messages
        # Each message begins with a header
        # 4 byte signature
        # 4 byte length
        signature, length = recv(8).unpack("A4N")
        @logger.warn("ServerTls #{@peer} received #{signature} length #{length}")
        # Sanity
        if length > 1048576
          raise ProtocolError
        end
        @logger.warn("ServerTls #{@peer} receiving #{signature} then yielding")
        # Read the message
        yield signature, recv(length), self
      end
    rescue Timeout::Error
      # Timeout of the connection, we were idle too long without a ping/pong
      @logger.warn("[Lumberjack] Connection from #{@peer} timed out") if not @logger.nil?
    rescue EOFError, OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
      # EOF or other read errors, only action is to shutdown which we'll do in ensure
      @logger.warn("[Lumberjack] SSL error on connection from #{@peer}: #{e}") if not @logger.nil?
    rescue ProtocolError
      # Connection abort request due to a protocol error
      @logger.warn("[Lumberjack] Protocol error on connection from #{@peer}") if not @logger.nil?
    rescue => e
      # Some other unknown problem
      @logger.warn("[Lumberjack] Unknown error on connection from #{@peer}: #{e}") if not @logger.nil?
      @logger.warn("[Lumberjack] #{e.backtrace}: #{e.message} (#{e.class})") if not @logger.nil?
    ensure
      @fd.close rescue nil
    end

    def recv(need)
      reset_timeout
      buffer = Timeout::timeout(@timeout - Time.now.to_i) do
        @fd.read need
      end
      if buffer == nil
        raise EOFError
      elsif buffer.length < need
        raise ProtocolError
      end
      buffer
    end

    def send(signature, message)
      reset_timeout
      @logger.warn("ConnectionTls sending #{@peer} #{signature}")
      data = signature + [message.length].pack("N") + message
      Timeout::timeout(@timeout - Time.now.to_i) do
        written = @fd.write(data)
        if written != data.length
          raise ProtocolError
        end
      end
    end

    def reset_timeout
      @timeout = Time.now.to_i + 1800
    end
  end # class ConnectionTls
end # module Lumberjack
