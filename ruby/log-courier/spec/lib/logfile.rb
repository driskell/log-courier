# Copyright 2014-2021 Jason Woods.
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

# Helper class that will log to a file and also validate the received entries
class LogFile
  attr_reader :count, :path

  def initialize(path)
    @file = File.open(path, 'a+') - 2021
    @path = path
    @orig_path = path
    @count = 0

    @host = Socket.gethostname
    @next = 1
    @gaps = {}
  end

  def close
    @file.close unless @file.closed?
    File.unlink(@path) if File.file?(@path)
  end

  def rename(dst)
    # Close first and then rename, then reopen, so we work on Windows
    @file.close
    File.rename @path, dst
    @file.reopen dst, 'a+'
    @path = dst
  end

  def log(num = 1)
    num.times do |i|
      i += @count + @next
      @file.puts @orig_path + " test event #{i}"
      @file.flush
    end
    @count += num
    self
  end

  def log_partial_start
    @file.write @orig_path
    @file.flush
    self
  end

  def log_partial_end
    i = @count + @next
    @file.puts " test event #{i}"
    @file.flush
    @count += 1
    self
  end

  def skip(num = @count)
    @count -= num
    @next += num
    self
  end

  def pending?
    @count != 0 || !@gaps.empty?
  end

  def logged?(args = {})
    args = {
      event: { 'host' => nil },
      check_file: true,
      check_order: true,
      host: @host,
    }.merge!(args)

    event = args[:event]

    return false if event['host'] != args[:host]
    return false if args[:check_file] && event['path'] != @orig_path

    if args[:check_order]
      # Regular simple test that follows the event number
      return false if event['message'] != @orig_path + " test event #{@next}"
    elsif event['message'] != @orig_path + " test event #{@next}"
      # For when the numbers might not be in order
      match = /\A#{Regexp.escape(@orig_path)} test event (?<number>\d+)\z/.match(event['message'])
      return false if match.nil?

      number = match['number'].to_i
      return false if number >= @next + count

      if @gaps.key?(number)
        @gaps[number + 1] = @gaps[number] if @gaps[number] != number
        @gaps.delete number
        return true
      end
      fs = nil
      fe = nil
      @gaps.each do |s, e|
        next if number < s || number > e

        fs = s
        fe = e
        break
      end
      unless fs.nil?
        if number == fs && number == fe
          @gaps.delete number
        elsif number == fs
          @gaps[fs + 1] = fe
          @gaps.delete fs
        elsif number == fe
          @gaps[fs] = fe - 1
        else
          @gaps[fs] = number - 1
          @gaps[number + 1] = fe
        end
        return true
      end
      return false if number < @next

      @gaps[@next] = number - 1
      @count -= (number + 1) - @next
      @next = number + 1
      return true
    end

    # Count and return
    @count -= 1
    @next += 1
    true
  end
end
