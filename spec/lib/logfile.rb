# Helper class that will log to a file and also validate the received entries
class LogFile
  attr_reader :count
  attr_reader :path

  def initialize(path)
    @file = File.open(path, "a+")
    @path = path
    @orig_path = path
    @count = 0

    @host = Socket.gethostname
    @next = 1
  end

  def close
    @file.close if not @file.closed?
    File.unlink(@path) if File.exists?(@path)
  end

  def rename(dst)
    File.rename @path, dst
    @path = dst
  end

  def log(num=1)
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

  def skip(num=@count)
    @count -= num
    @next += num
    self
  end

  def has_pending?
    @count != 0
  end

  def logged?(event, check_file=true)
    return false if event["host"] != @host
    return false if check_file and event["file"] != @orig_path
    return false if event["message"] != @orig_path + " test event #{@next}"
    @count -= 1
    @next += 1
    true
  end
end
