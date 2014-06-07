# encoding: utf-8
# TODO: Were these needed? Output doesn't seem to need them
#require "logstash/inputs/base"
#require "logstash/namespace"

# Receive events using the lumberjack protocol
class LogStash::Inputs::Lumberjack < LogStash::Inputs::Base
  config_name "lumberjack"
  milestone 1

  default :codec, "plain"

  # The IP address to listen on
  config :host, :validate => :string, :default => "0.0.0.0"

  # The port to listen on
  config :port, :validate => :number, :required => true

  # The transport type to use
  config :transport, :validate => :string, :default => "tls"

  # SSL certificate to use
  config :ssl_certificate, :validate => :path, :required => true

  # SSL key to use
  config :ssl_key, :validate => :path, :required => true

  # SSL key passphrase to use
  config :ssl_key_passphrase, :validate => :password

  # Whether or not to verify client certificates
  config :ssl_verify, :validate => :boolean, :default => false

  # When verifying client certificates, also trust those signed by the system's default CA bundle
  config :ssl_verify_default_ca, :validate => :boolean, :default => false

  # CA certificate to use when verifying client certificates
  config :ssl_verify_ca, :validate => :path

  public
  def register
    @logger.info("Starting lumberjack input listener", :address => "#{@host}:#{@port}")

    require "lumberjack/server"
    @lumberjack = Lumberjack::Server.new(
      :logger                => @logger,
      :address               => @host,
      :port                  => @port,
      :transport             => @transport,
      :ssl_certificate       => @ssl_certificate,
      :ssl_key               => @ssl_key,
      :ssl_key_passphrase    => @ssl_key_passphrase,
      :ssl_verify            => @ssl_verify,
      :ssl_verify_default_ca => @ssl_verify_default_ca,
      :ssl_verify_ca         => @ssl_verify_ca,
    )
  end

  public
  def run(output_queue)
    # TODO: How do we handle Logstash shutdown? An unknown exception raised in run will trigger it but what waits for it?
    # TODO: Check this is correct with the new protocol results
    @lumberjack.run do |e|
      # TODO: Should we even bother with line?
      @codec.decode(e.delete("line")) do |event|
        decorate event
        output_queue << event
      end
    end
  end
end
