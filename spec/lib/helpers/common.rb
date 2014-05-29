require "lumberjack/server"

# Common helpers for testing both ruby client and the forwarder
shared_context "Helpers" do
  before :all do
    @ssl_cert = File.open(File.join(TEMP_PATH, "ssl_cert"), "w")
    @ssl_key = File.open(File.join(TEMP_PATH, "ssl_key"), "w")
    @ssl_csr = File.open(File.join(TEMP_PATH, "ssl_csr"), "w")

    # Generate the ssl key
    system("openssl genrsa -out #{@ssl_key.path} 1024")
    system("openssl req -new -key #{@ssl_key.path} -batch -out #{@ssl_csr.path}")
    system("openssl x509 -req -days 365 -in #{@ssl_csr.path} -signkey #{@ssl_key.path} -out #{@ssl_cert.path}")
  end

  after :all do
    [@ssl_cert, @ssl_key, @ssl_csr].each do |f|
      File.unlink(f.path) if File.exists?(f.path)
    end
  end

  before :each do
    # When we add a file we log it here, so after we can remove them
    @files = []

    # Reset server for each test
    @server = Lumberjack::Server.new(
      :ssl_certificate => @ssl_cert.path,
      :ssl_key => @ssl_key.path
    )

    @event_queue = Queue.new
    @server_thread = Thread.new do
      @server.run do |event|
        @event_queue << event
      end
    end
  end

  after :each do
    # Remove any files we created for the test
    @files.each do |f|
      f.close
    end

    @files = []
  end

  # A helper that creates a new log file
  def create_log(type=LogFile, path=nil)
    path ||= File.join(TEMP_PATH, "logs", "log-" + @files.length.to_s)

    # Return a new file for testing, and log it for cleanup afterwards
    f = type.new(path)
    @files.push(f)
    return f
  end

  # Rename a log file and create a new one in its place
  def rotate(f, prefix="")
    old_name = f.path

    if prefix == ""
      new_name = f.path + "r"
    else
      new_name = File.join(File.dirname(f.path), prefix + File.basename(f.path) + "r")
    end

    f.rename new_name

    create_log(f.class, old_name)
  end

  def wait_for_events(total)
    waited = 0
    last_total = 0
    while @event_queue.length != total and waited < EVENT_WAIT_COUNT
      if @event_queue.length != last_total
        waited = 0
        last_total = @event_queue.length
      end
      sleep(EVENT_WAIT_TIME)
      waited += 1
    end

    @event_queue.length.should == total
  end

  def receive_and_check(check_file: true)
    # Quick check of the total events we are expecting - but allow time to receive them
    total = @files.inject(0) do |sum, f|
      sum + f.count
    end

    wait_for_events total

    while @event_queue.length > 0
      e = @event_queue.pop
      f = @files.find do |f|
        f.logged?(e, check_file)
      end
      f.should_not be_nil, "Event received not recognised: #{e}"
    end
  end
end
