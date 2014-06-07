require "logger"
require "lumberjack/server"

# Common helpers for testing both ruby client and the forwarder
shared_context "Helpers" do
  before :all do
    @transport = "tls"

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

    @event_queue = Queue.new

    @servers = Hash.new
    @server_threads = Hash.new

    start_server
  end

  after :each do
    # Remove any files we created for the test
    @files.each do |f|
      f.close
    end

    @files = []

    shutdown_server
  end

  # A helper that starts a lumberjack server
  def start_server(id: "__default__", transport: nil)
    logger = Logger.new(STDOUT)
    logger.progname = "Server #{id}"
    logger.level = Logger::DEBUG

    raise "Server already initialised" if @servers.has_key?(id)

    if transport.nil?
      transport = @transport
    end

    # Reset server for each test
    @servers[id] = Lumberjack::Server.new(
      :transport => transport,
      :ssl_certificate => @ssl_cert.path,
      :ssl_key => @ssl_key.path,
      :logger => logger
    )

    @server_threads[id] = Thread.new do
      begin
        @servers[id].run do |event|
          @event_queue << event
        end
      rescue
      end
    end
  end

  # A helper to shutdown a lumberjack server
  def shutdown_server(id=nil)
    if id.nil?
      id = @servers.keys
    else
      id = [id]
    end
    id.each do |id|
      @server_threads[id].raise Lumberjack::ShutdownSignal
      @server_threads[id].join
      @server_threads.delete id
      @servers.delete id
    end
  end

  # A helper to get the port a server is bound to
  def server_port(id='__default__')
    @servers[id].port
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

  def receive_and_check(total: nil, check_file: true, &block)
    # Quick check of the total events we are expecting - but allow time to receive them
    if total.nil?
      total = @files.inject(0) do |sum, f|
        sum + f.count
      end
    end

    waited = 0
    while total > 0 and waited < EVENT_WAIT_COUNT
      if @event_queue.length != 0
        waited = 0
        while @event_queue.length != 0
          e = @event_queue.pop
          if block != nil
            block.call e
          else
            f = @files.find do |f|
              if f.has_pending?
                f.logged?(e, check_file)
              else
                false
              end
            end
            expect(f).to_not be_nil, "Event received not recognised: #{e}"
          end
          total -= 1
        end
        next
      end
      sleep(EVENT_WAIT_TIME)
      waited += 1
    end

    expect(total).to eq 0
  end
end
