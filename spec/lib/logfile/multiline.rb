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

class LogFile
  # Multiline replaced the event checks so they match multiline events
  class Multiline < LogFile
    def log(num = 1)
      num.times do |i|
        i += @count + @next
        # This pattern is chosen specifically as we can match it with all the different match types
        @file.puts 'BEGIN ' + @path + " test event #{i}"
        @file.puts " line 2 of test event #{i}"
        @file.puts " line 3 of test event #{i}"
        @file.puts ' END of test event'
      end
      @file.flush
      @count += num
      self
    end

    def skip_one
      @count -= 1
    end

    def logged?(event: event, check_file: true, check_order: true)
      return false if event['host'] != @host
      return false if check_file && event['path'] != @path
      return false if event['message'] != 'BEGIN ' + @path + " test event #{@next}" + $/ + " line 2 of test event #{@next}" + $/ + " line 3 of test event #{@next}" + $/ + ' END of test event'
      @count -= 1
      @next += 1
      true
    end
  end
end
