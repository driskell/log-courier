# encoding: utf-8

# Send events using the lumberjack protocol
class LogStash::Outputs::Lumberjack < LogStash::Outputs::Base
  config_name "lumberjack"
  milestone 1

  # The list of addresses lumberjack should send to
  config :hosts, :validate => :array, :required => true

  # The port to connect to
  config :port, :validate => :number, :required => true

  # CA certificate for validation of the server
  config :ssl_ca, :validate => :path, :required => true

  # Client SSL certificate to use
  config :ssl_certificate, :validate => :path

  # Client SSL key to use
  config :ssl_key, :validate => :path

  # Maximum number of events to spool before forcing a flush
  config :spool_size, :validate => :number, :default => 1024

  # Maximum time to wait for a full spool before forcing a flush
  config :idle_timeout, :validate => :number, :default => 5

  public
  def register
    require 'lumberjack/client'
    connect

    @client = Lumberjack::Client.new(
      :addresses          => @hosts,
      :port               => @port, 
      :ssl_ca             => @ssl_ca,
      :ssl_certificate    => @ssl_certificate,
      :ssl_key            => @ssl_key,
      :spool_size         => @spool_size,
      :idle_timeout       => @idle_timeout,
    )

    @codec.on_event do |event|
      @client.publish event
    end
  end

  public
  def receive(event)
    return unless output?(event)
    if event == LogStash::SHUTDOWN
      @client.shutdown
      finished
      return
    end
    @codec.encode event
  end
end
