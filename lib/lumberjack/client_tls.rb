require "socket"
require "thread"
require "openssl"

module Lumberjack
  class ClientTls
    def initialize(options={})
      @options = {
        :logger             => nil,
        :port               => 0,
        :addresses          => [],
        :ssl_ca             => nil,
        :ssl_certificate    => nil,
        :ssl_key            => nil,
        :ssl_key_passphrase => nil,
      }.merge(options)

      @logger = @options[:logger]

      c = 0
      [:ssl_certificate, :ssl_key].each do
        c = c + 1
      end
      raise "You must specify both ssl_certificate and ssl_key if either is specified in Lumberjack::Client.new(...)" if c == 1

      raise "You must specify a port in Lumberjack::Client.new(...)" if @options[:port] == 0
      raise "You must specify at least one address in Lumberjack::Client.new(...)" if @options[:addresses].empty? == 0
      raise "You must specity the ssl ca certificate in Lumberjack::Client.new(...)" if @options[:ssl_ca].nil?
    end

    def connect(io_control)
      begin
        tls_connect
      rescue ClientShutdownSignal
        raise
      rescue => e
        # TODO: Make this configurable
        sleep 5
        retry
      end

      @send_q = SizedQueue.new 1
      @send_paused = false

      @send_thread = Thread.new do
        run_send io_control
      end
      @recv_thread = Thread.new do
        run_recv io_control
      end
    end

    def disconnect
      @send_thread.raise ClientShutdownSignal
      @send_thread.join
      @recv_thread.raise ClientShutdownSignal
      @recv_thread.join
    end

    def run_send(io_control)
      # Ask for something to send
      io_control << ["S"]

      # If paused, we still accept message to send, but we don't release "S" to ask for more
      # As soon as we resume we then release "S" to ask for more
      paused = false

      while true
        # Wait for data and send when we get it
        message = @send_q.pop

        # A nil is a pause/resume
        if message.nil?
          if paused
            paused = false
            io_control << ["S"]
          else
            paused = true
            next
          end
        else
          # Ask for more to send while we send this one
          if not paused
            io_control << ["S"]
          end

          @ssl_client.write message
        end
      end
    rescue OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
      @logger.warn("[LumberjackClientTLS] SSL write error: #{e}") if not @logger.nil?
      io_control << ["F"]
    rescue ClientShutdownSignal
      # Just shutdown
    rescue => e
      @logger.warn("[LumberjackClientTLS] Unknown SSL write error: #{e}") if not @logger.nil?
      @logger.debug("[LumberjackClientTLS] #{e.backtrace}: #{e.message} (#{e.class})") if not @logger.nil? and @logger.debug?
      io_control << ["F"]
    end

    def run_recv(io_control)
      while true
        # Grab a header
        header = @ssl_client.read(8)
        raise EOFError if header.nil?

        # Decode signature and length
        signature, length = header.unpack("A4N")

        if length > 1048576
          # Too big raise error
          @logger.warn("[LumberjackClientTLS] Invalid message: data too big (#{length})") if not @logger.nil?
          io_control << ["F"]
          break
        end

        # Read remainder
        message = @ssl_client.read(length)

        # Pass through to receive
        io_control << ["R", signature, message]
      end
    rescue OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
      @logger.warn("[LumberjackClientTLS] SSL read error: #{e}") if not @logger.nil?
      io_control << ["F"]
    rescue EOFError
      @logger.warn("[LumberjackClientTLS] Connection closed by server") if not @logger.nil?
      io_control << ["F"]
    rescue ClientShutdownSignal
      # Just shutdown
    rescue => e
      @logger.warn("[LumberjackClientTLS] Unknown SSL read error: #{e}") if not @logger.nil?
      @logger.debug("[LumberjackClientTLS] #{e.backtrace}: #{e.message} (#{e.class})") if not @logger.nil? and @logger.debug?
      io_control << ["F"]
    end

    def send(signature, message)
      # Add to send queue
      @send_q << [signature, message.length].pack("A4N") + message
    end

    def pause_send
      if not @send_paused
        @send_paused = true
        @send_q << nil
      end
    end

    def send_paused?
      return @send_paused
    end

    def resume_send
      if @send_paused
        @send_paused = false
        @send_q << nil
      end
    end

    def tls_connect
      begin
        @logger.info("[LumberjackClientTLS] Connecting to #{@options[:addresses][0]}:#{@options[:port]}") if not @logger.nil?
        tcp_socket = TCPSocket.new(@options[:addresses][0], @options[:port])

        ssl = OpenSSL::SSL::SSLContext.new

        if not @options[:ssl_certificate].nil?
          ssl.cert = OpenSSL::X509::Certificate.new(File.read(@options[:ssl_certificate]))
          ssl.key = OpenSSL::PKey::RSA.new(File.read(@options[:ssl_key]), @options[:ssl_key_passphrase])
        end

        cert_store = OpenSSL::X509::Store.new
        cert_store.add_file(@options[:ssl_ca])
        ssl.cert_store = cert_store
        ssl.verify_mode = OpenSSL::SSL::VERIFY_PEER|OpenSSL::SSL::VERIFY_FAIL_IF_NO_PEER_CERT

        @ssl_client = OpenSSL::SSL::SSLSocket.new(tcp_socket)

        socket = @ssl_client.connect
        @logger.info("[LumberjackClientTLS] Connected successfully") if not @logger.nil?

        socket
      rescue OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
        @logger.warn("[LumberjackClientTLS] Connection to #{@options[:addresses][0]}:#{@options[:port]} failed: #{e}") if not @logger.nil?
      rescue ClientShutdownSignal
        # Just shutdown
      rescue => e
        @logger.warn("[LumberjackClientTLS] Unknown connection failure to #{@options[:addresses][0]}:#{@options[:port]}: #{e}") if not @logger.nil?
        @logger.debug("[LumberjackClientTLS] #{e.backtrace}: #{e.message} (#{e.class})") if not @logger.nil? and @logger.debug?
      end
    end
  end
end
