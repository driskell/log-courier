require 'logger'
require 'lumberjack/server'

# Common helpers for testing both ruby client and the forwarder
shared_context 'Helpers' do
  before :all do
    @transport = 'tls'

    @ssl_cert = File.open(File.join(TEMP_PATH, 'ssl_cert'), 'w')
    @ssl_key = File.open(File.join(TEMP_PATH, 'ssl_key'), 'w')
    @ssl_csr = File.open(File.join(TEMP_PATH, 'ssl_csr'), 'w')

    # Generate the ssl key
    system("openssl genrsa -out #{@ssl_key.path} 1024")
    system("openssl req -new -key #{@ssl_key.path} -batch -out #{@ssl_csr.path}")
    system("openssl x509 -req -days 365 -in #{@ssl_csr.path} -signkey #{@ssl_key.path} -out #{@ssl_cert.path}")
  end

  after :all do
    [@ssl_cert, @ssl_key, @ssl_csr].each do |f|
      File.unlink(f.path) if File.file?(f.path)
    end
  end

  before :each do
    # When we add a file we log it here, so after we can remove them
    @files = []

    @event_queue = Queue.new

    @servers = {}
    @server_counts = {}
    @server_threads = {}

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
  def start_server(id: '__default__', transport: nil)
    logger = Logger.new(STDOUT)
    logger.progname = "Server #{id}"
    logger.level = Logger::DEBUG

    raise 'Server already initialised' if @servers.key?(id)

    transport = @transport if transport.nil?

    # Reset server for each test
    @servers[id] = Lumberjack::Server.new(
      :transport        => transport,
      :ssl_certificate  => @ssl_cert.path,
      :ssl_key          => @ssl_key.path,
      :curve_secret_key => '1XQgjDjkw?YP=$f61HKe%g+AEbe<VZt%{#8).G0j',
      :logger           => logger
    )

    @server_counts[id] = 0
    @server_threads[id] = Thread.new do
      begin
        @servers[id].run do |event|
          @server_counts[id] += 1
          @event_queue << event
        end
      rescue Lumberjack::ShutdownSignal
        0
      end
    end
  end

  # A helper to shutdown a lumberjack server
  def shutdown_server(which = nil)
    if which.nil?
      which = @servers.keys
    else
      which = [which]
    end
    which.each do |id|
      @server_threads[id].raise Lumberjack::ShutdownSignal
      @server_threads[id].join
      @server_threads.delete id
      @server_counts.delete id
      @servers.delete id
    end
  end

  # A helper to get the port a server is bound to
  def server_port(id = '__default__')
    @servers[id].port
  end

  # A helper to get number of events received on the server
  def server_count(id = '__default__')
    @server_counts[id]
  end

  # A helper that creates a new log file
  def create_log(type = LogFile, path = nil)
    path ||= File.join(TEMP_PATH, 'logs', 'log-' + @files.length.to_s)

    # Return a new file for testing, and log it for cleanup afterwards
    f = type.new(path)
    @files.push(f)
    f
  end

  # Rename a log file and create a new one in its place
  def rotate(f, prefix = '')
    old_name = f.path

    if prefix == ''
      new_name = f.path + 'r'
    else
      new_name = File.join(File.dirname(f.path), prefix + File.basename(f.path) + 'r')
    end

    f.rename new_name

    create_log(f.class, old_name)
  end

  def receive_and_check(total: nil, check_file: true, check_order: true, &block)
    # Quick check of the total events we are expecting - but allow time to receive them
    if total.nil?
      total = @files.reduce(0) do |sum, f|
        sum + f.count
      end
    end

    orig_total = total

    waited = 0
    while total > 0 && waited < EVENT_WAIT_COUNT
      if @event_queue.length == 0
        sleep(EVENT_WAIT_TIME)
        waited += 1
        next
      end

      waited = 0
      while @event_queue.length != 0
        e = @event_queue.pop
        if block.nil?
          found = @files.find do |f|
            next unless f.pending?
            f.logged?(event: e, check_file: check_file, check_order: check_order)
          end
          expect(found).to_not be_nil, "Event received not recognised: #{e}"
        else
          block.call e
        end
        total -= 1
      end
    end

    # Fancy calculation to give a nice "expected" output of expected num of events
    expect(orig_total - total).to eq orig_total
  end
end
