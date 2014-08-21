# encoding: utf-8

# Copyright 2014 Jason Woods.
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
# The majority of the code is taken from Ruby's SizedQueue implementation.
#
class EventQueue < Queue
  class TimeoutError < StandardError; end

  #
  # Creates a fixed-length queue with a maximum size of +max+.
  #
  def initialize(max)
    raise ArgumentError, "queue size must be positive" unless max > 0
    @max = max
    @enque_cond = ConditionVariable.new
    @num_enqueue_waiting = 0
    super()
  end

  #
  # Returns the maximum size of the queue.
  #
  def max
    @max
  end

  #
  # Sets the maximum size of the queue.
  #
  def max=(max)
    raise ArgumentError, "queue size must be positive" unless max > 0

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
    max
  end

  #
  # Pushes +obj+ to the queue.  If there is no space left in the queue, waits
  # until space becomes available, up to a maximum of +timeout+ seconds.
  #
  def push(obj, timeout = nil)
    unless timeout.nil?
      start = Time.now
    end
    Thread.handle_interrupt(RuntimeError => :on_blocking) do
      @mutex.synchronize do
        while true
          break if @que.length < @max
          @num_enqueue_waiting += 1
          begin
            @enque_cond.wait @mutex, timeout
          ensure
            @num_enqueue_waiting -= 1
          end
          raise Timeout if !timeout.nil? and Time.now - start >= timeout
        end

        @que.push obj
        @cond.signal
      end
      self
    end
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
    retval = _pop_timeout *args
    @mutex.synchronize do
      if @que.length < @max
        @enque_cond.signal
      end
    end
    retval
  end

  #
  # Retrieves data from the queue.  If the queue is empty, the calling thread is
  # suspended until data is pushed onto the queue or, if set, +timeout+ seconds
  # passes.  If +timeout+ is 0, the thread isn't suspended, and an exception is
  # raised.
  #
  def _pop_timeout(timeout = nil)
    Thread.handle_interrupt(StandardError => :on_blocking) do
      @mutex.synchronize do
        loop do
          return @que.shift unless @que.empty?
          raise ThreadError, 'queue empty' if timeout == 0
          begin
            @num_waiting += 1
            @cond.wait @mutex, timeout
          ensure
            @num_waiting -= 1
          end
        end
      end
    end
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
  # Returns the number of threads waiting on the queue.
  #
  def num_waiting
    @num_waiting + @num_enqueue_waiting
  end
end
