# encoding: utf-8

# Copyright 2014 Jason Woods.
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

require 'thread'
require 'log-courier/zmq_qpoll'

module LogCourier
  # ZMQ transport implementation for the server
  class ServerZmq
    class ZMQError < StandardError; end

    class << self
      @print_zmq_versions = false

      def print_zmq_versions(logger)
        return if @print_zmq_versions || logger.nil?

        libversion = LibZMQ.version
        libversion = "#{libversion[:major]}.#{libversion[:minor]}.#{libversion[:patch]}"

        logger.info 'libzmq', :version => libversion
        logger.info 'ffi-rzmq-core', :version => LibZMQ::VERSION
        logger.info 'ffi-rzmq', :version => ZMQ.version

        @print_zmq_versions = true
      end
    end

    attr_reader :port

    def initialize(options = {})
      @options = {
        logger:           nil,
        transport:        'zmq',
        port:             0,
        address:          '0.0.0.0',
        curve_secret_key: nil,
        max_packet_size:  10_485_760,
        peer_recv_queue:  10,
      }.merge!(options)

      @logger = @options[:logger]

      self.class.print_zmq_versions @logger

      if @options[:transport] == 'zmq'
        fail "input/courier: Transport 'zmq' requires libzmq version >= 4" unless LibZMQ.version4?

        fail 'input/courier: \'curve_secret_key\' is required' if @options[:curve_secret_key].nil?

        fail 'input/courier: \'curve_secret_key\' must be a valid 40 character Z85 encoded string' if @options[:curve_secret_key].length != 40 || !z85validate(@options[:curve_secret_key])
      end

      begin
        @context = ZMQ::Context.new
        # Router so we can send multiple responses
        @socket = @context.socket(ZMQ::ROUTER)

        if @options[:transport] == 'zmq'
          rc = @socket.setsockopt(ZMQ::CURVE_SERVER, 1)
          fail 'setsockopt CURVE_SERVER failure: ' + ZMQ::Util.error_string unless ZMQ::Util.resultcode_ok?(rc)

          rc = @socket.setsockopt(ZMQ::CURVE_SECRETKEY, @options[:curve_secret_key])
          fail 'setsockopt CURVE_SECRETKEY failure: ' + ZMQ::Util.error_string unless ZMQ::Util.resultcode_ok?(rc)
        end

        bind = 'tcp://' + @options[:address] + (@options[:port] == 0 ? ':*' : ':' + @options[:port].to_s)
        rc = @socket.bind(bind)
        fail 'failed to bind at ' + bind + ': ' + rZMQ::Util.error_string unless ZMQ::Util.resultcode_ok?(rc)

        # Lookup port number that was allocated in case it was set to 0
        endpoint = ''
        rc = @socket.getsockopt(ZMQ::LAST_ENDPOINT, endpoint)
        fail 'getsockopt LAST_ENDPOINT failure: ' + ZMQ::Util.error_string unless ZMQ::Util.resultcode_ok?(rc) && %r{\Atcp://(?:.*):(?<endpoint_port>\d+)\0\z} =~ endpoint
        @port = endpoint_port.to_i

        if @options[:port] == 0
          @logger.warn 'Ephemeral port allocated', :transport => @options[:transport], :port => @port unless @logger.nil?
        end
      rescue => e
        raise "input/courier: Failed to initialise: #{e}"
      end

      # TODO: Implement workers option by receiving on a ROUTER and proxying to a DEALER, with workers connecting to the DEALER

      # TODO: Make this send queue configurable?
      @send_queue = EventQueue.new 2
      @factory = ClientFactoryZmq.new(@options, @send_queue)

      # Setup poller
      @poller = ZMQPoll::ZMQPoll.new(@context)
      @poller.register_socket @socket, ZMQ::POLLIN
      @poller.register_queue_to_socket @send_queue, @socket

      # Register a finaliser that sets @context to nil
      # This allows us to detect the JRuby bug where during "exit!" finalisers
      # are run but threads are not killed - which leaves us in a situation of
      # a terminated @context (it has a terminate finalizer) and an IO thread
      # looping retries
      # JRuby will still crash and burn, but at least we don't spam STDOUT with
      # errors
      ObjectSpace.define_finalizer(self, Proc.new do
        @context = nil
      end)
    end

    def run(&block)
      loop do
        begin
          @poller.poll(5_000) do |socket, r, w|
            next if socket != @socket
            next if !r

            receive &block
          end
        rescue ZMQPoll::ZMQError => e
          # Detect JRuby bug
          fail e if @context.nil?
          @logger.warn e, :hint => 'ZMQ recv_string failure' unless @logger.nil?
          next
        rescue ZMQPoll::TimeoutError
          # We'll let ZeroMQ manage reconnections and new connections
          # There is no point in us doing any form of reconnect ourselves
          next
        end
      end
      return
    rescue ShutdownSignal
      # Shutting down
      @logger.warn 'Server shutting down' unless @logger.nil?
      return
    rescue StandardError, NativeException => e
      # Some other unknown problem
      @logger.warn e, :hint => 'Unknown error, shutting down' unless @logger.nil?
      raise e
    ensure
      @poller.shutdown
      @factory.shutdown
      @socket.close
      @context.terminate
    end

    private

    def z85validate(z85)
      # ffi-rzmq does not implement decode - but we want to validate during startup
      decoded = FFI::MemoryPointer.from_string(' ' * (8 * z85.length / 10))
      ret = LibZMQ.zmq_z85_decode decoded, z85
      return false if ret.nil?
      true
    end

    def receive(&block)
      # Try to receive a message
      data = []
      rc = @socket.recv_strings(data, ZMQ::DONTWAIT)
      unless ZMQ::Util.resultcode_ok?(rc)
        fail ZMQError, 'recv_string error: ' + ZMQ::Util.error_string if ZMQ::Util.errno != ZMQ::EAGAIN
      end

      # Save the source information that appears before the null messages
      source = []
      source.push data.shift until data.length == 0 || data[0] == ''

      if data.length == 0
        @logger.warn 'Invalid message: no data', :source_length => source.length unless @logger.nil?
        return
      elsif data.length == 1
        @logger.warn 'Invalid message: empty data', :source_length => source.length unless @logger.nil?
        return
      end

      # Drop the null message separator
      data.shift

      if data.length != 1
        @logger.warn 'Invalid message: multipart unexpected', :source_length => source.length, :data_length => data.length unless @logger.nil?
        if !@logger.nil? && @logger.debug?
          i = 0
          parts = {}
          data.each do |msg|
            i += 1
            parts[i] = "#{part.length}:[#{msg[0..31].gsub(/[^[:print:]]/, '.')}]"
          end
          @logger.debug 'Data', parts
        end
        return
      end

      @factory.deliver source, data.first, &block
      return
    end
  end

  class ClientFactoryZmq
    attr_reader :options
    attr_reader :send_queue

    def initialize(options, send_queue)
      @options = options
      @logger = @options[:logger]

      @send_queue = send_queue
      @index = {}
      @client_threads = {}
      @mutex = Mutex.new
    end

    def shutdown
      # Stop other threads from try_drop collisions
      client_threads = @mutex.synchronize do
        client_threads = @client_threads
        @client_threads = {}
        client_threads
      end

      client_threads.each_value do |thr|
        thr.raise ShutdownSignal
      end

      client_threads.each_value(&:join)
      return
    end

    def deliver(source, data, &block)
      # Find the handling thread
      # We separate each source into threads so that each thread can respond
      # with partial ACKs if we hit a slow down
      # If we processed in a single thread, we'd only be able to respond to
      # a single client with partial ACKs
      @mutex.synchronize do
        index = @index
        source.each do |identity|
          index[identity] = {} if !index.key?(identity)
          index = index[identity]
        end

        if !index.key?('')
          source_str = source.map do |s|
            s.each_byte.map do |b|
              b.to_s(16).rjust(2, '0')
            end
          end.join

          @logger.info 'New source', :source => source_str unless @logger.nil?

          # Create the client and associated thread
          client = ClientZmq.new(self, source, source_str) do
            try_drop(source)
          end

          thread = Thread.new do
            client.run &block
          end

          @client_threads[thread] = thread

          index[''] = {
            'client' => client,
            'thread' => thread,
          }
        end

        # Existing thread, throw on the queue, if not enough room drop the message
        index['']['client'].push data, 0
      end
      return
    end

    private

    def try_drop(source, source_str)
      # This is called when a client goes idle, to cleanup resources
      # We may tie this into zmq monitor
      @mutex.synchronize do
        index = @index
        parents = []
        source.each do |identity|
          if !index.key?(identity)
            @logger.warn 'Unknown idle source failed to shutdown', :source => source_str unless @logger.nil?
            break
          end
          parents.push [index, identity]
          index = index[identity]
        end

        if !index.key?('')
          @logger.warn 'Unknown idle source failed to shutdown', :source => source_str unless @logger.nil?
          break
        end

        # Don't allow drop if we have messages in the queue
        if index['']['client'].length != 0
          @logger.warn 'Failed idle source shutdown as message queue is not empty', :source => source_str unless @logger.nil?
          return false
        end

        @logger.info 'Idle source shutting down', :source => source_str unless @logger.nil?

        # Delete the entry
        @client_threads.delete(index['']['thread'])
        index.delete('')

        # Cleanup orphaned leafs
        parents.reverse_each do |path|
          path[0].delete(path[1]) if path[0][path[1]].length == 0
        end
      end

      return true
    end
  end

  class ClientZmq < EventQueue
    def initialize(factory, source, source_str, &try_drop)
      @factory = factory
      @logger = @factory.options[:logger]
      @send_queue = @factory.send_queue
      @source = source
      @source_str = source_str
      @try_drop = try_drop

      # Setup the queue for receiving events to process
      super @factory.options[:peer_recv_queue]
    end

    def run(&block)
      loop do
        begin
          # TODO: Make timeout configurable?
          data = self.pop(30)
          recv(data, &block)
        rescue TimeoutError
          # Try to clean up resources - if we fail, new messages have arrived
          retry if !@try_drop.call(@source)
          break
        end
      end
      return
    rescue ShutdownSignal
      # Shutting down
      @logger.info 'Source shutting down', :source => @source_str unless @logger.nil?
      return
    rescue StandardError, NativeException => e
      # Some other unknown problem
      @logger.warn e, :hint => 'Unknown error, connection aborted', :source => @source_str unless @logger.nil?
      raise e
    end

    def send(signature, message)
      data = signature + [message.length].pack('N') + message
      @send_queue.push @source + ['', data]
      return
    end

    private

    def recv(data)
      if data.length < 8
        @logger.warn 'Invalid message: not enough data', :data_length => data.length, :source => @source_str unless @logger.nil?
        return
      end

      # Unpack the header
      signature, length = data.unpack('A4N')

      # Verify length
      if data.length - 8 != length
        @logger.warn 'Invalid message: data has invalid length', :data_length => data.length - 8, :encoded_length => length, :source => @source_str unless @logger.nil?
        return
      elsif length > @factory.options[:max_packet_size]
        @logger.warn 'Invalid message: packet too large', :size => length, :max_packet_size => @options[:max_packet_size], :source => @source_str unless @logger.nil?
        return
      end

      # Yield the parts
      yield signature, data[8, length], self
      return
    end
  end
end
