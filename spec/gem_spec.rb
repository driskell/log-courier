require "logger"
require "timeout"
require "lib/common"

require "lumberjack/client"

describe "logstash-forwarder gem" do
  include_context "Helpers"

  before :all do
    @host = Socket.gethostname
  end

  def startup
        puts "Startup #{@server.port}"
    logger = Logger.new(STDOUT)
    logger.level = Logger::DEBUG

    # Reset server for each test
    @client = Lumberjack::Client.new(
      :ssl_certificate => @ssl_cert.path,
      :addresses => "127.0.0.1",
      :port => @server.port,
      :logger => logger
    )
        puts "Started"
  end

  it "should send and receive events" do
    startup

    # Allow 60 seconds
    Timeout::timeout(60) do
      5000.times do |i|
        puts "Iteration #{i}"
        @client.write "message" => "gem line test #{i}", "host" => @host, "file" => "gemfile.log"
      end
    end

    # Receive and check
    wait_for_events 5000

    i = 0
    while @event_queue.length > 0
      e = @event_queue.pop
      e["line"].should == "gem line test #{i}"
      e["host"].should == @host
      e["file"].should == "gemfile.log"
      i += 1
    end
  end
end
