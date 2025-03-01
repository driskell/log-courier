# Copyright 2014-2021 Jason Woods and Contributors.
#
# This file is a modification of code from Ruby.
# Ruby is copyrighted free software by Yukihiro Matsumoto <matz@netlab.jp>.
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

#
# This is a queue implementation dervied from SizedQueue, but with a timeout.
#
# It is significantly faster than using SizedQueue wrapped in Timeout::timeout
# because it uses mutex.sleep, whereas Timeout::timeout actually starts another
# thread that waits and then raises exception or has to be stopped on exiting
# the block.
#
# The majority of the code is taken from Ruby's SizedQueue<Queue implementation.
#
module LogCourier
  # EventQueue
  class EventQueue
    #
    # Creates a fixed-length queue with a maximum size of +max+.
    #
    def initialize(max)
      raise ArgumentError, 'queue size must be positive' unless max.positive?

      @max = max
      @enque_cond = ConditionVariable.new
      @num_enqueue_waiting = 0

      @que = []
      @num_waiting = 0
      @mutex = Mutex.new
      @cond = ConditionVariable.new
    end

    #
    # Returns the maximum size of the queue.
    #
    attr_reader :max

    #
    # Sets the maximum size of the queue.
    #
    def max=(max)
      raise ArgumentError, 'queue size must be positive' unless max.positive?

      @mutex.synchronize do
        if max <= @max
          @max = max
        else
          diff = max - @max
          @max = max
          diff.times do
            @enque_cond.signal
          end
        end
      end
    end

    #
    # Pushes +obj+ to the queue.  If there is no space left in the queue, waits
    # until space becomes available, up to a maximum of +timeout+ seconds.
    #
    def push(obj, timeout = nil)
      start = Time.now unless timeout.nil?
      @mutex.synchronize do
        loop do
          break if @que.length < @max

          @num_enqueue_waiting += 1
          begin
            @enque_cond.wait @mutex, timeout
          ensure
            @num_enqueue_waiting -= 1
          end

          raise TimeoutError if !timeout.nil? && Time.now - start >= timeout
        end

        @que.push obj
        @cond.signal
      end
      self
    end

    #
    # Alias of push
    #
    alias << push

    #
    # Alias of push
    #
    alias enq push

    #
    # Retrieves data from the queue and runs a waiting thread, if any.
    #
    def pop(*args)
      retval = pop_timeout(*args)
      @mutex.synchronize do
        @enque_cond.signal if @que.length < @max
      end
      retval
    end

    #
    # Alias of pop
    #
    alias shift pop

    #
    # Alias of pop
    #
    alias deq pop

    #
    # Returns +true+ if the queue is empty.
    #
    def empty?
      @mutex.synchronize do
        return @que.empty?
      end
    end

    #
    # Removes all objects from the queue.
    #
    def clear
      @mutex.synchronize do
        @que.clear
        @enque_cond.signal if @que.length < @max
      end
      self
    end

    #
    # Returns the length of the queue.
    #
    def length
      @mutex.synchronize do
        return @que.length
      end
    end

    #
    # Alias of length.
    #
    alias size length

    #
    # Returns the number of threads waiting on the queue.
    #
    def num_waiting
      @mutex.synchronize do
        return @num_waiting + @num_enqueue_waiting
      end
    end

    private

    #
    # Retrieves data from the queue.  If the queue is empty, the calling thread is
    # suspended until data is pushed onto the queue or, if set, +timeout+ seconds
    # passes.  If +timeout+ is 0, the thread isn't suspended, and an exception is
    # raised.
    #
    def pop_timeout(timeout = nil)
      start = Time.now unless timeout.nil?
      @mutex.synchronize do
        loop do
          return @que.shift unless @que.empty?
          raise TimeoutError if !timeout.nil? && timeout.zero?

          begin
            @num_waiting += 1
            @cond.wait @mutex, timeout
          ensure
            @num_waiting -= 1
          end
          raise TimeoutError if !timeout.nil? && Time.now - start >= timeout
        end
      end
      nil
    end
  end
end
