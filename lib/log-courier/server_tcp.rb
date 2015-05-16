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

require 'cabin'
require 'iomultiplex'
require 'multi_json'
require 'openssl'
require 'socket'
require 'thread'
require 'zlib'

module LogCourier
  # TLS transport implementation for server
  class TCPTransport
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

      @options[:logger] = Cabin::Channel.new unless @options[:logger]
      @logger = @options[:logger]

      if @options[:transport] == 'tls'
        [:ssl_certificate, :ssl_key].each do |k|
          fail "input/courier: '#{k}' is required" if @options[k].nil?
        end

        if @options[:ssl_verify] and (!@options[:ssl_verify_default_ca] && @options[:ssl_verify_ca].nil?)
          fail 'input/courier: Either \'ssl_verify_default_ca\' or \'ssl_verify_ca\' must be specified when ssl_verify is true'
        end

        ssl_ctx = OpenSSL::SSL::SSLContext.new

        # Disable SSLv2 and SSLv3
        # Call set_params first to ensure options attribute is there (hmmmm?)
        ssl_ctx.set_params

        # Modify the default options to ensure SSLv2 and SSLv3 is disabled
        # This retains any beneficial options set by default in the current Ruby implementation
        ssl_ctx.options |= OpenSSL::SSL::OP_NO_SSLv2 if defined?(OpenSSL::SSL::OP_NO_SSLv2)
        ssl_ctx.options |= OpenSSL::SSL::OP_NO_SSLv3 if defined?(OpenSSL::SSL::OP_NO_SSLv3)

        # Set the certificate file
        ssl_ctx.cert = OpenSSL::X509::Certificate.new(File.read(@options[:ssl_certificate]))
        ssl_ctx.key = OpenSSL::PKey::RSA.new(File.read(@options[:ssl_key]), @options[:ssl_key_passphrase])

        if @options[:ssl_verify]
          cert_store = OpenSSL::X509::Store.new

          # Load the system default certificate path to the store
          cert_store.set_default_paths if @options[:ssl_verify_default_ca]

          if File.directory?(@options[:ssl_verify_ca])
            cert_store.add_path @options[:ssl_verify_ca]
          else
            cert_store.add_file @options[:ssl_verify_ca]
          end

          ssl_ctx.cert_store = cert_store

          ssl_ctx.verify_mode = OpenSSL::SSL::VERIFY_PEER | OpenSSL::SSL::VERIFY_FAIL_IF_NO_PEER_CERT
        end
      else
        ssl_ctx = nil
      end

      @message_queue = Queue.new

      @listener = IOMultiplex::Multiplexer.new(logger: @logger, id: 'Listener')
      #@pool = IOMultiplex::MultiplexerPool.new(logger: @logger, id: 'Pool', parent: @listener, num_workers: 4)
      @server = IOMultiplex::TCPListener.new(@options[:address], @options[:port]) do |socket|
        TCPClient.new(socket, @options, @message_queue, ssl_ctx)
      end
      @listener.add @server, 'Listener'

      # Query the port in case the port number is '0'
      # TCPServer#addr == [ address_family, port, address, address ]
      @port = @server.addr[1]
      if @options[:port] == 0
        @logger.warn 'Ephemeral port allocated', :transport => @options[:transport], :port => @port
      end

      # Load the json adapter
      @json_adapter = MultiJson.adapter.instance
      @json_options = { raw: true }
    rescue => e
      raise e, "input/courier: Failed to initialise: #{e}"
    end # def initialize

    def run(event_queue)
      @listen_thread = Thread.new do
        begin
          @listener.run
        rescue Exception, NativeException => e
          puts e.inspect, e.backtrace
        end
      end

      @receiver_thread = Thread.new do
        receiver event_queue
      end
    end

    def shutdown
      return unless @listen_thread
      @logger.debug 'Shutdown request received'
      @listener.shutdown
      @listen_thread.join
      @logger.debug 'Listener thread shutdown'
      #@pool.shutdown
      @logger.debug 'Multiplexer pool shutdown'
      @message_queue << nil
      @receiver_thread.join
      @logger.debug 'Receiver thread shutdown'
    end

    private

    def receiver(event_queue)
      loop do
        message = @message_queue.pop
        return unless message

        nonce, data, client = message

        process_jdat nonce, data, client, event_queue
      end
    end

    def process_jdat(nonce, data, client, event_queue)
      # Now we have the data, aim to respond within 5 seconds
      ack_timeout = Time.now.to_i + 5

      # The remainder of the message is the compressed data block
      data = StringIO.new Zlib::Inflate.inflate(data)

      # Message now contains JSON encoded events
      # They are aligned as [length][event]... so on
      # We acknowledge them by their 1-index position in the stream
      # A 0 sequence acknowledgement means we haven't processed any yet
      sequence = 0
      events = []
      length_buf = ''
      data_buf = ''
      loop do
        ret = data.read 4, length_buf
        if ret.nil?
          # Finished!
          break
        elsif length_buf.length < 4
          @logger.warn 'JDAT length extraction failed', id: client.id, ret: ret, length_length: length_buf.length
          client.shutdown
          return
        end

        length = length_buf.unpack('N').first

        # Extract message
        ret = data.read length, data_buf
        if ret.nil? or data_buf.length < length
          @logger.warn 'JDAT data extraction failed', id: client.id, ret: ret, data_length: data_buf.length
          client.shutdown
          return
        end

        data_buf.force_encoding('utf-8')

        # Ensure valid encoding
        unless data_buf.valid_encoding?
          data_buf.chars.map do |c|
            c.valid_encoding? ? c : "\xEF\xBF\xBD"
          end
        end

        # Decode the JSON
        begin
          event = @json_adapter.load(data_buf, @json_options)
        rescue MultiJson::ParseError => e
          @logger.warn e, :hint => 'JSON parse failure, falling back to plain-text', id: client.id
          event = { 'message' => data_buf }
        end

        # Add peer fields?
        client.add_fields event

        # Queue the event
        begin
          event_queue.push event, [0, ack_timeout - Time.now.to_i].max
        rescue TimeoutError
          # Full pipeline, partial ack
          # NOTE: comm.send can raise a Timeout::Error of its own
          @logger.warn 'Partially acknowledging message', id: client.id, :nonce => nonce_str(nonce), :sequence => sequence #if @logger.debug?
          return unless client.ackn(nonce, sequence)
          ack_timeout = Time.now.to_i + 5
          retry
        end

        sequence += 1
      end

      # Acknowledge the full message
      # NOTE: comm.send can raise a Timeout::Error
      @logger.warn 'Acknowledging message', id: client.id, :nonce => nonce_str(nonce), :sequence => sequence #if @logger.debug?
      return unless client.ackn(nonce, sequence, true)
    rescue Zlib::DataError => e
      @logger.warn e, :hint => 'Decompression failure', id: client.id
      client.shutdown
    end

    def nonce_str(nonce)
      nonce.each_byte.map do |b|
        b.to_s(16).rjust(2, '0')
      end.join
    end
  end

  class TempTimer < IOMultiplex::Timer
    def initialize(obj)
      super()
      @obj = obj
    end

    def timer
      @obj.send :timeout
    end
  end

  class TCPClient < IOMultiplex::SSLUpgradingIO
    def initialize(socket, options, message_queue, ssl_ctx)
      super socket, 'rw'
      @options = options
      @logger = options[:logger]
      @message_queue = message_queue
      @mutex = Mutex.new
      @peer_fields = []
      @timer = TempTimer.new(self)
      @pending = []

      start_ssl ssl_ctx if ssl_ctx

      @logger.info 'New connection', :id => id
    end

    def handshake_completed
      if @options[:add_peer_fields]
        @peer_fields['peer'] = peer
        if @options[:transport] == 'tls' and not peer_cert_cn.nil?
          @peer_fields['peer_ssl_cn'] = peer_cert_cn
        end
      end
    end

    def add_fields(event)
      event.merge! @peer_fields if @peer_fields.length != 0
    end

    def attached(multiplexer, logger)
      super multiplexer, logger

      # TODO: Configurable timeout for handshake?
      set_timeout 30
      @state = :header
    end

    def handshake_completed
      # TODO: Configurable timeout
      set_timeout 3600
    end

    def process
      loop do
        break unless __send__(@state)
      end

      return
    end

    def exception(e)
      # TODO: More information in logs
      @logger.warn 'Socket exception', :exception => e, :state => @state, :now => Time.now.to_i, :timer => @timer.time.to_i
    end

    def eof
      if @state == :header
        @logger.info 'Connection closed'
      else
        @logger.info 'Remote side disconnect unexectedly', :state => @state
      end
    end

    def ackn(nonce, sequence, completed=false)
      @multiplexer.callback do
        process_ackn nonce, sequence, completed
      end
      return true
    end

    def shutdown
      @multiplexer.callback do
        force_close if @attached
      end

      return
    end

    def timeout
      return unless @attached

      unless handshake_completed?
        @logger.warn 'Handshake timeout'
        force_close
        return
      end

      # Are we pending?
      if @pending.length != 0
        keepalive
        return
      end

      # Timeout! Close connection
      @logger.warn 'Read timeout', :state => @state
      force_close
      return
    end

    private

    def header
      # Read header
      # 4 byte signature
      # 4 byte length
      @signature, @length = read(8).unpack('A4N')

      # Sanity
      if @length > @options[:max_packet_size]
        @logger.warn 'packet too large', :length => @length, :max_package_size => @options[:max_packet_size]
        force_close
        return
      end

      # Next state
      @state = :message
      true
    end

    def message
      case @signature
      when 'JDAT'
        message_jdat
      when 'PING'
        message_ping
      else
        unknown_message
      end
    end

    def message_jdat
      if @length < 17
        read(@length)
        @logger.warn 'JDAT data too small', :length => @length
        @state = :header
        return
      end

      @nonce = read(16)
      @length -= 16

      # Push onto pending queue and start keepalive if we haven't already
      @logger.warn 'JDAT message received', :nonce => nonce_str(@nonce), :length => @length
      set_timeout 8 if @pending.length == 0
      @pending.push @nonce

      @state = :message_jdat_data
      true
    end

    def message_jdat_data
      data = read(@length)

      @logger.warn 'JDAT message queueing', :nonce => nonce_str(@nonce)
      @message_queue << [@nonce, data, self]

      @state = :header

      # Only hold 2 messages in memory
      # (This means 2*max_packet_size memory approx per client)
      # This is so we can immediately begin the processing of the next message
      # once this one finishes. It also ensures that we start partial acks for
      # the next message immediately rather than waiting for the data to be
      # received. (In tests, waiting for the data before beginning partial acks
      # caused excessive timeouts as Log Courier itself starts timing
      # immediately)
      if @pending.length > 1
        pause
        false
      else
        true
      end
    end

    def message_ping
      if @length != 0
        @logger.warn 'invalid PING message size', :length => @length
        force_close
        return
      end

      write_message 'PONG', ''
      # If in idle mode, reset timeout
      if @pending.length == 0
        # TODO: configurable timeout?
        set_timeout 3600
      end

      @state = :header
      true
    end

    def unknown_message
      if comm.peer.nil?
        @logger.warn 'Unknown message received'
      else
        @logger.warn 'Unknown message received'
      end
      # Don't kill a client that sends a bad message
      # Just reject it and let it send it again, potentially to another server
      write_message '????', ''
      @state = :header
      true
    end

    def keepalive
      # Send a fake partial ack with sequence 0
      @logger.warn 'Keepalive', :state => @state, :nonce => nonce_str(@pending[0]), :sequence => 0 #if @logger.debug?
      write_message 'ACKN', [@pending[0], 0].pack('a*N')

      # Set another timeout
      set_timeout 8
      return
    end

    def process_ackn(nonce, sequence, completed=false)
      return unless @attached

      @logger.warn 'Writing acknowledgement', :nonce => nonce_str(nonce), :sequence => sequence, :completed => completed.inspect

      # If the sequence is complete, remove the pending log
      if completed
        @pending.shift
        # Resume idle timeout if no longer any pending work
        if @pending.length == 0
          # TODO: Configurable timeout?
          set_timeout 3600
        else
          set_timeout 8
        end

        # Resume processing
        resume
      else
        # Our payload is now getting handled! Let's stop faking partial acks
        # and wait for the processor to give us progress
        @multiplexer.remove_timer @timer
      end

      write_message 'ACKN', [nonce, sequence].pack('A*N')
      return
    end

    def write_message(signature, data)
      write signature + [data.length].pack('N') + data
      return
    end

    def set_timeout(timeout)
      @multiplexer.add_timer @timer, Time.now + timeout
      return
    end

    def nonce_str(nonce)
      nonce.each_byte.map do |b|
        b.to_s(16).rjust(2, '0')
      end.join
    end
  end
end
