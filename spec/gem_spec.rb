require "lib/common"

require "lumberjack/client"

describe "logstash-forwarder gem" do
  include_context "Helpers"

  before :all do
    @host = Socket.gethostname
  end

  def startup
    # Reset server for each test
    @client = Lumberjack::Client.new(
      :ssl_certificate => @ssl_cert.path,
      :addresses => "127.0.0.1",
      :port => @server.port
    )
  end

  it "should send and receive events" do
    startup

    5000.times do |i|
      @client.write "line" => "gem line test #{i}", "host" => @host, "file" => "gemfile.log"
    end

    # Receive and check
    wait_for_events 5000

    i = 0
    while @event_queue.length > 0
      e = @event_queue.pop
      e["message"].should == "gem line test #{i}"
      e["host"].should == @host
      e["file"].should == "gemfile.log"
      i += 1
    end
  end
end
