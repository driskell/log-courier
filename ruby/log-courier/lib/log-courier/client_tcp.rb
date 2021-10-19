# encoding: utf-8

# Copyright 2014-2019 Jason Woods and Contributors.
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
  # TLS transport implementation
  class ClientTcp
    def initialize(options = {})
      @options = {
        logger:             nil,
        transport:         'tls',
        ssl_ca:             nil,
        ssl_certificate:    nil,
        ssl_key:            nil,
        ssl_key_passphrase: nil,
        min_tls_version:    1.2,
      }.merge!(options)

      @logger = @options[:logger]

      [:port, :ssl_ca].each do |k|
        fail "output/courier: '#{k}' is required" if @options[k].nil?
      end

      if @options[:transport] == 'tls'
        c = 0
        [:ssl_certificate, :ssl_key].each do
          c += 1
        end
        fail 'output/courier: \'ssl_certificate\' and \'ssl_key\' must be specified together' if c == 1
      end
    end

    def connect(io_control)
      loop do
        begin
          if tls_connect
            handshake(io_control)
            break
          end
        rescue ShutdownSignal
          raise
        end

        # TODO: Make this configurable
        sleep 5
      end

      @send_q = SizedQueue.new 1
      @send_paused = false

      @send_thread = Thread.new do
        begin
          run_send io_control
        rescue ShutdownSignal
        rescue StandardError, NativeException => e # Can remove NativeException after 9.2.14.0 JRuby
          @logger.warn e, :hint => 'Unknown write error' unless @logger.nil?
          io_control << ['F']
        end
      end
      @recv_thread = Thread.new do
        begin
          run_recv io_control
        rescue ShutdownSignal
        rescue StandardError, NativeException => e # Can remove NativeException after 9.2.14.0 JRuby
          @logger.warn e, :hint => 'Unknown read error' unless @logger.nil?
          io_control << ['F']
        end
      end
      return
    end

    def handshake(io_control)
      @ssl_client.write ['HELO', 8, 0, 1, 11, 0, 'RYLC'].pack('A4NCCCCA4')

      signature, data = receive
      if signature != 'VERS'
        fail "Unexpected message during handshake: #{signature}" if signature != '????'
        @vers = Protocol.parseHeloVers('')
        @logger.info 'Remote does not support protocol handshake', :server_version => @vers[:client_version] unless @logger.nil?
        return
      end

      @vers = Protocol.parseHeloVers(data)
      @logger.info 'Remote identified', :server_version => @vers[:client_version] unless @logger.nil?
    rescue ShutdownSignal
    rescue StandardError, NativeException => e # Can remove NativeException after 9.2.14.0 JRuby
      @logger.warn e, :hint => 'Unknown write error' unless @logger.nil?
      io_control << ['F']
      return
    end

    def disconnect
      @send_thread.raise ShutdownSignal
      @send_thread.join
      @recv_thread.raise ShutdownSignal
      @recv_thread.join
      return
    end

    def send(signature, message)
      # Add to send queue
      @send_q << [signature, message.length].pack('A4N') + message
      return
    end

    def pause_send
      return if @send_paused
      @send_paused = true
      @send_q << nil
      return
    end

    def send_paused?
      @send_paused
    end

    def resume_send
      if @send_paused
        @send_paused = false
        @send_q << nil
      end
      return
    end

    private

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
      return
    rescue OpenSSL::SSL::SSLError => e
      @logger.warn 'SSL write error', :error => e.message unless @logger.nil?
      io_control << ['F']
      return
    rescue IOError, Errno::ECONNRESET => e
      @logger.warn 'Write error', :error => e.message unless @logger.nil?
      io_control << ['F']
      return
    end

    def run_recv(io_control)
      loop do
        signature, message = receive

        # Pass through to receive
        io_control << ['R', signature, message]
      end
      return
    rescue OpenSSL::SSL::SSLError => e
      @logger.warn 'SSL read error', :error => e.message unless @logger.nil?
      io_control << ['F']
      return
    rescue IOError, Errno::ECONNRESET => e
      @logger.warn 'Read error', :error => e.message unless @logger.nil?
      io_control << ['F']
      return
    rescue EOFError
      @logger.warn 'Connection closed by server' unless @logger.nil?
      io_control << ['F']
      return
    end

    def receive
      # Grab a header
      header = @ssl_client.read(8)
      fail EOFError if header.nil?

      # Decode signature and length
      signature, length = header.unpack('A4N')

      if length > 1048576
        # Too big raise error
        fail IOError, 'Invalid message: data too big'
      end

      # Read remainder
      message = @ssl_client.read(length)

      return signature, message
    end

    def tls_connect
      # TODO: Implement random selection - and don't use separate :port - remember to update post_connection_check too
      address = @options[:addresses][0]
      port = @options[:port]

      @logger.info 'Connecting', :address => address, :port => port unless @logger.nil?

      begin
        tcp_socket = TCPSocket.new(address, port)

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
          fail 'Invalid min_tls_version - max is 1.3' if @options[:min_tls_version] > 1.3

          # Set the certificate file
          unless @options[:ssl_certificate].nil?
            ssl.cert = OpenSSL::X509::Certificate.new(File.read(@options[:ssl_certificate]))
            ssl.key = OpenSSL::PKey::RSA.new(File.read(@options[:ssl_key]), @options[:ssl_key_passphrase])
          end

          cert_store = OpenSSL::X509::Store.new
          cert_store.add_file(@options[:ssl_ca])
          ssl.cert_store = cert_store
          ssl.verify_mode = OpenSSL::SSL::VERIFY_PEER | OpenSSL::SSL::VERIFY_FAIL_IF_NO_PEER_CERT

          @ssl_client = OpenSSL::SSL::SSLSocket.new(tcp_socket, ssl)

          socket = @ssl_client.connect

          # Verify certificate
          socket.post_connection_check(address)
        else
          socket = tcp_socket.connect
        end

        # Add extra logging data now we're connected
        @logger['address'] = address
        @logger['port'] = port

        @logger.info 'Connected successfully', :ssl_version => @ssl_client.ssl_version unless @logger.nil?
        return true
      rescue OpenSSL::SSL::SSLError, IOError, Errno::ECONNRESET => e
        @logger.warn 'Connection failed', :error => e.message, :address => address, :port => port unless @logger.nil?
      rescue StandardError, NativeException => e # Can remove NativeException after 9.2.14.0 JRuby
        @logger.warn e, :hint => 'Unknown connection failure', :address => address, :port => port unless @logger.nil?
      end

      false
    end
  end
end
