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

require 'socket'
require 'thread'
require 'openssl'

module LogCourier
  # TLS transport implementation
  class ClientTls
    def initialize(options = {})
      @options = {
        :logger             => nil,
        :port               => nil,
        :addresses          => [],
        :ssl_ca             => nil,
        :ssl_certificate    => nil,
        :ssl_key            => nil,
        :ssl_key_passphrase => nil
      }.merge(options)

      @logger = @options[:logger]

      [:port, :ssl_ca].each do |k|
        raise "[LogCourierClientTLS] '#{k}' is required" if @options[k].nil?
      end

      raise '[LogCourierClientTLS] \'addresses\' must contain at least one address' if @options[:addresses].empty?

      c = 0
      [:ssl_certificate, :ssl_key].each do
        c += 1
      end

      raise '[LogCourierClientTLS] \'ssl_certificate\' and \'ssl_key\' must be specified together' if c == 1
    end

    def connect(io_control)
      begin
        tls_connect
      rescue ClientShutdownSignal
        raise
      rescue
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
      io_control << ['S']

      # If paused, we still accept message to send, but we don't release "S" to ask for more
      # As soon as we resume we then release "S" to ask for more
      paused = false

      loop do
        # Wait for data and send when we get it
        message = @send_q.pop

        # A nil is a pause/resume
        if message.nil?
          if paused
            paused = false
            io_control << ['S']
          else
            paused = true
            next
          end
        else
          # Ask for more to send while we send this one
          io_control << ['S'] unless paused

          @ssl_client.write message
        end
      end
    rescue OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
      @logger.warn("[LogCourierClientTLS] SSL write error: #{e}") unless @logger.nil?
      io_control << ['F']
    rescue ClientShutdownSignal
      # Just shutdown
    rescue => e
      @logger.warn("[LogCourierClientTLS] Unknown SSL write error: #{e}") unless @logger.nil?
      @logger.debug("[LogCourierClientTLS] #{e.backtrace}: #{e.message} (#{e.class})") unless @logger.nil? || !@logger.debug?
      io_control << ['F']
    end

    def run_recv(io_control)
      loop do
        # Grab a header
        header = @ssl_client.read(8)
        raise EOFError if header.nil?

        # Decode signature and length
        signature, length = header.unpack('A4N')

        if length > 1048576
          # Too big raise error
          @logger.warn("[LogCourierClientTLS] Invalid message: data too big (#{length})") unless @logger.nil?
          io_control << ['F']
          break
        end

        # Read remainder
        message = @ssl_client.read(length)

        # Pass through to receive
        io_control << ['R', signature, message]
      end
    rescue OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
      @logger.warn("[LogCourierClientTLS] SSL read error: #{e}") unless @logger.nil?
      io_control << ['F']
    rescue EOFError
      @logger.warn("[LogCourierClientTLS] Connection closed by server") unless @logger.nil?
      io_control << ['F']
    rescue ClientShutdownSignal
      # Just shutdown
    rescue => e
      @logger.warn("[LogCourierClientTLS] Unknown SSL read error: #{e}") unless @logger.nil?
      @logger.debug("[LogCourierClientTLS] #{e.backtrace}: #{e.message} (#{e.class})") unless @logger.nil? || !@logger.debug?
      io_control << ['F']
    end

    def send(signature, message)
      # Add to send queue
      @send_q << [signature, message.length].pack('A4N') + message
    end

    def pause_send
      return if @send_paused
      @send_paused = true
      @send_q << nil
    end

    def send_paused
      @send_paused
    end

    def resume_send
      if @send_paused
        @send_paused = false
        @send_q << nil
      end
    end

    def tls_connect
      @logger.info("[LogCourierClientTLS] Connecting to #{@options[:addresses][0]}:#{@options[:port]}") unless @logger.nil?
      tcp_socket = TCPSocket.new(@options[:addresses][0], @options[:port])

      ssl = OpenSSL::SSL::SSLContext.new

      unless @options[:ssl_certificate].nil?
        ssl.cert = OpenSSL::X509::Certificate.new(File.read(@options[:ssl_certificate]))
        ssl.key = OpenSSL::PKey::RSA.new(File.read(@options[:ssl_key]), @options[:ssl_key_passphrase])
      end

      cert_store = OpenSSL::X509::Store.new
      cert_store.add_file(@options[:ssl_ca])
      ssl.cert_store = cert_store
      ssl.verify_mode = OpenSSL::SSL::VERIFY_PEER | OpenSSL::SSL::VERIFY_FAIL_IF_NO_PEER_CERT

      @ssl_client = OpenSSL::SSL::SSLSocket.new(tcp_socket)

      socket = @ssl_client.connect
      @logger.info("[LogCourierClientTLS] Connected successfully") unless @logger.nil?

      socket
    rescue OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
      @logger.warn("[LogCourierClientTLS] Connection to #{@options[:addresses][0]}:#{@options[:port]} failed: #{e}") unless @logger.nil?
    rescue ClientShutdownSignal
      # Just shutdown
      0
    rescue => e
      @logger.warn("[LogCourierClientTLS] Unknown connection failure to #{@options[:addresses][0]}:#{@options[:port]}: #{e}") unless @logger.nil?
      @logger.debug("[LogCourierClientTLS] #{e.backtrace}: #{e.message} (#{e.class})") unless @logger.nil? || !@logger.debug?
    end
  end
end
