# Common helpers for testing the forwarder
shared_context "Helpers_LSF" do
  before :each do
    @config = File.open(File.join(TEMP_PATH, "config"), "w")

    @logstash_forwarder = nil
  end

  after :each do
    shutdown

    File.unlink(@config.path) if File.exists?(@config.path)

    # This is important or the state will not be clean for the following test
    File.unlink(".logstash-forwarder") if File.exists?(".logstash-forwarder")
  end

  def startup(config: nil, args: "", mode: "r")
    # A standard configuration when we don't want anything special
    if config == nil
      config = <<-config
      {
        "network": {
          "servers": [ "127.0.0.1:#{@server.port}" ],
          "ssl ca": "#{@ssl_cert.path}",
          "timeout": 15,
          "reconnect": 1
        },
        "files": [
          {
            "paths": [ "#{TEMP_PATH}/logs/log-*" ]
          }
        ]
      }
      config
    end

    if @config.closed?
      # Reopen the config and rewrite it
      @config.reopen(@config.path, "w")
    end
    @config.puts config
    @config.close

    # Start LSF
    @logstash_forwarder = IO.popen("build/bin/logstash-forwarder -config #{@config.path}" + (args.empty? ? "" : " " + args), mode)

    # Needs some time to startup - to ensure when we create new files AFTER this, they are not detected during startup
    sleep STARTUP_WAIT_TIME
  end

  def shutdown
    if not @logstash_forwarder.nil?
      # When shutdown is implemented this becomes TERM/QUIT
      Process::kill("KILL", @logstash_forwarder.pid)
      Process::wait(@logstash_forwarder.pid)
      @logstash_forwarder = nil
    end
  end
end
