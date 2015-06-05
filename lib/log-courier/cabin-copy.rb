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
      copy = Cabin::Channel.new
    else
      copy = Cabin::Channel.get(identifier)
    end

    @data.each_key do |k|
      copy[k] = @data[k]
    end

    @subscribers.each_key do |k|
      copy.subscribe @subscribers[k]
    end

    copy
  end

  public :fork
end
