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
  def copy
    logger = Cabin::Channel.new
    logger.level = @level
    @data.each_key do |k|
      logger[k] = @data[k]
    end
    @subscribers.each_key do |k|
      logger.subscribe @subscribers[k]
    end
    return logger
  end
end
