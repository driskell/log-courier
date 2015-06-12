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

# Patch to copy Cabin channels, maintaining existing subscriptions,
# level, and data. Allows the new channel to have additional data attached that
# will not impact the copied channel.
class Cabin::Channel
  @default_level = :info
  @level_config = Hash.new { |h,k| h[k] = @default_level }

  @old_channels = @channels
  @channels = Hash.new { |h,k| h[k] = Cabin::Channel.new(@level_config[k]) }
  @old_channels.each_key do |k|
    @channels[k] = @old_channels[k]
  end

  class << self
    # Sets the default logging level for all new Cabin::Channel instances
    attr_accessor :default_level

    # Sets a granular logging configuration for cabin channels created via ::get
    # and #fork
    #
    # Parameter must be a Hash of values, where the key is the identifier and
    # the value the required logging level
    #
    # When a Cabin::Channel is created via ::get, this configuration will be
    # checked for the identifier, and the specified logging level used as the
    # initial level. If the identifier is not found, the default logging level
    # is used instead.
    #
    # Similarly, when a Cabin::Channel is created via #fork, this configuration
    # will also be used to ascertain the logging level required.
    def level_config=(value)
      fail ArgumentError, 'Array expected' unless value.is_a?(Hash)
      value.each_key do |identifier|
        @level_config[identifier] = value[identifier].to_sym
      end
    end

    # Get a channel for a given identifier. If this identifier has never been
    # used, a new channel is created for it.
    # The default identifier is the application executable name.
    #
    # This is useful for using the same Cabin::Channel across your
    # entire application.
    #
    # If a block is given and a new channel is created, the block is called with
    # the new channel so it may be configured thread-safe.
    def get(identifier=$0, &block)
      return @channel_lock.synchronize do
        if block_given? and !@channels.has_key?(identifier)
          block.call @channels[identifier]
        end
        @channels[identifier]
      end
    end # def Cabin::Channel.get
  end # class << self

  # Create a new logging channel.
  alias_method :orig_initialize, :initialize
  def initialize(level=nil)
    orig_initialize
    @level = level || self.class.default_level
  end # def initialize

  # Fork a Cabin::Channel into a second Cabin::Channel.
  #
  # The log level, data, and subscriptions of the original Cabin::Channel will
  # be inherited by the new channel.
  #
  # If an identifier is given, the existing channel for that identifier is
  # returned. If a channel does not exist for the given identifier, the source
  # channel is forked and stored under that identifier for future calls.
  # Additionally, if a granular logging level has been specified for the
  # identifier via ::level_config=, that logging level will be used instead of
  # inheriting it from the source.
  #
  # Changing the log level and adding or removing data or subscriptions will not
  # affect the original channel.
  #
  # If an identifier is provided in the first parameter, and a log level for
  # that identifier has been configured via Cabin::Channel#level_config=, that
  # level will be used as the initial logging level instead.
  def fork(identifier=nil)
    if identifier.nil?
      forked = Cabin::Channel.new
      copy forked
      return forked
    end

    Cabin::Channel.get(identifier) do |created|
      copy created
    end
  end

  # Copy our data and subscriptions to another channel
  def copy(target)
    @data.each_key do |k|
      target[k] = @data[k]
    end

    @subscribers.each_key do |k|
      target.subscribe @subscribers[k]
    end
  end

  public :fork
end
