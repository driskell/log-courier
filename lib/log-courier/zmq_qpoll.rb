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

begin
	require 'ffi-rzmq'
	require 'ffi-rzmq/version'
	require 'ffi-rzmq-core/version'
rescue LoadError => e
	raise "ZMQPoll could not initialise: #{e}"
end

module ZMQPoll
	class ZMQError < StandardError; end
	class TimeoutError < StandardError; end

	class ZMQPoll
		def initialize(context, logger=nil)
			@logger = logger
			@context = context
			@poller = ZMQ::Poller.new
			@sockets = []
			@socket_to_socket = []
			@handlers = {}
			@queues = {}
		end

		def readables
			@poller.readables
		end

		def writables
			@poller.writables
		end

		def shutdown
			@queues.each_key do |queue|
				deregister_queue queue
			end

			@socket_to_socket.each do |socket|
				_close_socket_to_socket socket
			end

			@sockets.each do |socket|
				socket.close
			end
			return
		end

		def register_socket(socket, flags)
			@poller.register socket, flags
			return
		end

		def deregister_socket(socket)
			return if @handlers.key?(socket)

			@poller.delete socket
			return
		end

		def register_queue_to_socket(queue, socket)
			s2s_state = _create_socket_to_socket(socket)

			state = {
				state:    s2s_state,
				mutex:    Mutex.new,
				shutdown: false,
			}

			state[:thread] = Thread.new do
				loop do
					data = queue.pop
					break if data.nil?
					begin
						send s2s_state[:sender], data
					rescue TimeoutError
						state[:mutex].synchronize do
							break if state[:shutdown]
						end
						retry
					end
				end
			end

			@queues[queue] = state
			return
		end

		def deregister_queue(queue)
			return if !@queues.key?(queue)

			# Push nil so if we're idle we jump into action and exit
			# But also set shutdown to try so if we're mid-send and timeout, we exit
			@queues[queue][:mutex].synchronize do
				queue.push nil
				@queues[queue][:shutdown] = true
			end
			@queues[queue][:thread].join

			_close_socket_to_socket @queues[queue][:state]
			@queues.delete queue
			return
		end

		def create_socket_to_socket(socket)
			state = _create_socket_to_socket(socket)
			@socket_to_socket[state[:sender]] = state
			state[:sender]
		end

		def close_socket_to_socket(socket)
			return if !@socket_to_socket.include?(socket)
			state = @socket_to_socket[socket]
			@socket_to_socket.delete socket
			_close_socket_to_socket(state)
			return
		end

		def poll(timeout)
			if @poller.size == 0
				fail ZMQError, 'poll run called with zero socket/queues'
			end

			rc = @poller.poll(timeout)
			if rc == -1
				fail ZMQError, 'poll error: ' + ZMQ::Util.error_string
			end

			return if rc == 0

			ready = (@poller.readables|@poller.writables)

			ready.each do |socket|
				if @handlers.key?(socket)
					__send__ @handlers[socket][:callback], @handlers[socket]
				end

				yield socket, @poller.readables.include?(socket), @poller.writables.include?(socket)
			end

			return
		end

		private

		def _create_socket_to_socket(socket)
			receiver = @context.socket(ZMQ::PULL)
			fail ZMQError, 'socket creation error: ' + ZMQ::Util.error_string if receiver.nil?

			rc = receiver.bind("inproc://zmqpollreceiver-#{receiver.hash}")
			fail ZMQError, 'bind error: ' + ZMQ::Util.error_string if !ZMQ::Util.resultcode_ok?(rc)

			sender = @context.socket(ZMQ::PUSH)
			fail ZMQError, 'socket creation error: ' + ZMQ::Util.error_string if sender.nil?

			rc = sender.connect("inproc://zmqpollreceiver-#{receiver.hash}")
			fail ZMQError, 'bind error: ' + ZMQ::Util.error_string if !ZMQ::Util.resultcode_ok?(rc)

			state = {
				:callback => :handle_socket_to_socket,
				:sender   => sender,
				:receiver => receiver,
				:socket   => socket,
				:buffer   => nil,
				:send_ok  => false,
				:recv_ok  => false,
			}

			@poller.register receiver, ZMQ::POLLIN
			@poller.register socket, ZMQ::POLLOUT
			@handlers[receiver] = state
			@handlers[socket] = state

			@sockets.push sender

			state
		end

		def _close_socket_to_socket(state)
			@sockets.delete state[:sender]

			@poller.delete state[:receiver]
			@poller.delete state[:socket]

			state[:sender].close
			state[:receiver].close

			@handlers.delete state[:receiver]
			@handlers.delete state[:socket]

			return
		end

		def handle_socket_to_socket(state)
			state[:recv_ok] = @poller.readables.include?(state[:receiver]) || state[:recv_ok]
			state[:send_ok] = @poller.writables.include?(state[:socket]) || state[:send_ok]

			loop do
				if state[:send_ok] && !state[:buffer].nil?
					begin
						send state[:socket], state[:buffer]
					rescue TimeoutError
					end
					state[:buffer] = nil if state[:buffer].length == 0
					state[:send_ok] = false
				end

				break if !state[:recv_ok]

				if state[:recv_ok] && state[:buffer].nil?
					begin
						state[:buffer] = recv(state[:receiver])
					rescue TimeoutError
					end
					state[:recv_ok] = false
				end

				break if !state[:send_ok]
			end

			if state[:recv_ok]
				@poller.deregister state[:receiver], ZMQ::POLLIN
			else
				@poller.register state[:receiver], ZMQ::POLLIN
			end

			if state[:send_ok]
				@poller.deregister state[:socket], ZMQ::POLLOUT
			else
				@poller.register state[:socket], ZMQ::POLLOUT
			end

			return
		end

		def recv(socket)
			data = []

			poll_eagain(socket, ZMQ::POLLIN, 5) do
				# recv_strings appears to be safe, ZMQ documents that a client will either
				# receive 0 parts or all parts
				socket.recv_strings(data, ZMQ::DONTWAIT)
			end

			data
		end

		def send(socket, data)
			while data.length != 1
				send_part socket, data.shift, true
			end
			send_part socket, data.shift
			return
		end

		def send_part(socket, data, more=false)
			poll_eagain(socket, ZMQ::POLLOUT, 5) do
				# Try to send a message but never block
				# We could use send_strings but it is vague on if ZMQ can return an
				# error midway through sending parts...
				socket.send_string(data, (more ? ZMQ::SNDMORE : 0) | ZMQ::DONTWAIT)
			end

			return
		end

		def poll_eagain(socket, flag, timeout, &block)
			poller = nil
			timeout = Time.now.to_i + timeout
			loop do
				rc = block.call()
				break if ZMQ::Util.resultcode_ok?(rc)
				if ZMQ::Util.errno != ZMQ::EAGAIN
					fail ZMQError, 'message receive failed: ' + ZMQ::Util.error_string if flag == ZMQ::POLLIN
					fail ZMQError, 'message send failed: ' + ZMQ::Util.error_string
				end

				# Wait for send to become available, handling timeouts
				if poller.nil?
					poller = ZMQ::Poller.new
					poller.register socket, flag
				end

				while poller.poll(1_000) == 0
					# Using this inner while triggers pollThreadEvents in JRuby which checks for Thread.raise immediately
					fail TimeoutError while Time.now.to_i >= timeout
				end
			end
			return
		end
	end
end
