# Helper class that will log to a file and also validate the received entries
class LogFile
  attr_reader :count
  attr_reader :path

  def initialize(path)
    @file = File.open(path, 'a+')
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
    File.rename @path, dst
    @path = dst
  end

  def log(num = 1)
    num.times do |i|
      i += @count + @next
      @file.puts @orig_path + " test event #{i}"
    end
    @file.flush
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

  def logged?(event: event, check_file: true, check_order: true)
    return false if event['host'] != @host
    return false if check_file && event['file'] != @orig_path

    if check_order
      # Regular simple test that follows the event number
      return false if event['message'] != @orig_path + " test event #{@next}"
    else
      # For when the numbers might not be in order
      if event['message'] != @orig_path + " test event #{@next}"
        match = /\A#{Regexp.escape(@orig_path)} test event (?<number>\d+)\z/.match(event['message'])
        return false if match.nil?
        number = match['number'].to_i
        return false if number >= @next + count
        if @gaps.key?(number)
          if @gaps[number] != number
            @gaps[number + 1] = @gaps[number]
          end
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
    end

    # Count and return
    @count -= 1
    @next += 1
    true
  end
end
