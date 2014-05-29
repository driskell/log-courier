require "lib/common"
require "lib/helpers/lsf"

describe "logstash-forwarder" do
  include_context "Helpers"
  include_context "Helpers_LSF"

  it "should follow stdin" do
    startup mode: "w", config: <<-config
    {
      "network": {
        "servers": [ "127.0.0.1:#{@server.port}" ],
        "ssl ca": "#{@ssl_cert.path}",
        "timeout": 15,
        "reconnect": 1
      },
      "files": [
        {
          "paths": [ "-" ]
        }
      ]
    }
    config

    5000.times do |i|
      @logstash_forwarder.puts "stdin line test #{i}"
    end

    # Receive and check
    wait_for_events 5000

    host = Socket.gethostname
    i = 0
    while @event_queue.length > 0
      e = @event_queue.pop
      e["line"].should == "stdin line test #{i}"
      e["host"].should == host
      e["file"].should == "-"
      i += 1
    end
  end

  it "should follow a file from the end" do
    # Hide lines in the file - this makes sure we start at the end of the file
    f = create_log.log(50).skip

    startup

    f.log 5000

    # Receive and check
    receive_and_check
  end

  it "should follow a file from the beginning with parameter -from-beginning=true" do
    # Hide lines in the file - this makes sure we start at the beginning of the file
    f = create_log.log(50)

    startup args: "-from-beginning=true"

    f.log 5000

    # Receive and check
    receive_and_check
  end

  it "should follow a slowly-updating file" do
    startup

    f = create_log

    100.times do |i|
      f.log 50

      # Start fast, then go slower after 80% of the events
      if i > 80
        sleep 0.2
      end
    end

    # Receive and check
    receive_and_check
  end

  it "should follow multiple files and resume them when restarted" do
    startup

    f1 = create_log
    f2 = create_log
    5000.times do
      f1.log
      f2.log
    end

    # Receive and check
    receive_and_check

    # Now restart logstash
    shutdown

    # From beginning makes testing this easier - without it we'd need to create lines inbetween shutdown and start and verify them which is more work
    startup args: "-from-beginning=true"

    f1 = create_log
    f2 = create_log
    5000.times do
      f1.log
      f2.log
    end

    # Receive and check
    receive_and_check
  end

  it "should start newly created files found after startup from beginning and not the end" do
    # Create a file and hide it
    f1 = create_log.log(5000)
    path = f1.path
    f1.rename File.join(File.dirname(path), "hide-" + File.basename(path))

    startup

    f2 = create_log.log(5000)

    # Throw the file back with all the content already there
    # We can't just create a new one, it might pick it up before we write
    f1.rename path

    # Receive and check
    receive_and_check
  end

  it "should handle incomplete lines in buffered logs by waiting for a line end" do
    startup

    f = create_log

    1000.times do |i|
      if (i + 100) % 500 == 0
        # Make 2 events where we pause for >10s before adding new line, this takes us past eof_timeout
        f.log_partial_start
        sleep 15
        f.log_partial_end
      else
        f.log
      end
    end

    # Receive and check
    receive_and_check
  end

  it "should handle log rotation and resume correctly" do
    startup

    f1 = create_log.log 100

    # Receive and check
    receive_and_check

    # Rotate f1 - this renames it and returns a new file same name as original f1
    f2 = rotate(f1)

    # Write to both
    f1.log 5000
    f2.log 5000

    # Receive and check
    receive_and_check

    # Restart
    shutdown
    startup

    # Write some more
    f1.log 5000
    f2.log 5000

    # Receive and check - but not file as it will be different now
    receive_and_check check_file: false
  end

  it "should handle log rotation and resume correctly even if rotated file moves out of scope" do
    startup

    f1 = create_log.log 100

    # Receive and check
    receive_and_check

    # Rotate f1 - this renames it and returns a new file same name as original f1
    # But prefix it so it moves out of scope
    f2 = rotate(f1)

    # Write to both - but a bit more to the out of scope
    f1.log 5000
    f2.log 5000
    f1.log 5000

    # Receive and check
    receive_and_check

    # Restart
    shutdown
    startup

    # Write some more but remember f1 should be out of scope
    f1.log(5000).skip 5000
    f2.log 5000

    # Receive and check - but not file as it will be different now
    receive_and_check check_file: false
  end

  it "should handle log rotation and resume correctly even if rotated file updated" do
    startup

    f1 = create_log.log 100

    # Receive and check
    receive_and_check

    # Rotate f1 - this renames it and returns a new file same name as original f1
    f2 = rotate(f1)

    # Write to both
    f1.log 5000
    f2.log 5000

    # Make the last update go to f1 (the rotated file)
    # This can throw up an edge case we used to fail
    sleep 10
    f1.log 5000

    # Receive and check
    receive_and_check

    # Restart
    shutdown
    startup

    # Write some more
    f1.log 5000
    f2.log 5000

    # Receive and check - but not file as it will be different now
    receive_and_check check_file: false
  end

  it "should handle log rotation during startup resume" do
    startup

    f1 = create_log.log 100

    # Receive and check
    receive_and_check

    # Stop
    shutdown

    # Rotate f1 - this renames it and returns a new file same name as original f1
    f2 = rotate(f1)

    # Write to both
    f1.log 5000
    f2.log(5000).skip 5000

    # Start again
    startup

    # Receive and check - but not file as it will be different now
    receive_and_check check_file: false
  end

  it "should resume harvesting a file that reached dead time but changed again" do
    startup config: <<-config
    {
      "network": {
        "servers": [ "127.0.0.1:#{@server.port}" ],
        "ssl ca": "#{@ssl_cert.path}",
        "timeout": 15,
        "reconnect": 1
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ],
          "dead time": "5s"
        }
      ]
    }
    config

    f1 = create_log.log(5000)

    # Receive and check
    receive_and_check

    # Let dead time occur
    sleep 15

    # Write again
    f1.log(5000)

    # Receive and check
    receive_and_check
  end

  it "should prune deleted files from registrar state" do
    # We use dead time to make sure the harvester stops, as file deletion is only acted upon once the harvester stops
    startup config: <<-config
    {
      "network": {
        "servers": [ "127.0.0.1:#{@server.port}" ],
        "ssl ca": "#{@ssl_cert.path}",
        "timeout": 15,
        "reconnect": 1
      },
      "files": [
        {
          "paths": [ "#{TEMP_PATH}/logs/log-*" ],
          "dead time": "5s"
        }
      ]
    }
    config

    # Write lines
    f1 = create_log.log(5000)
    f2 = create_log.log(5000)

    # Receive and check
    receive_and_check

    # Grab size of the saved state - sleep to ensure it was saved
    sleep 1
    s = File::Stat.new(".logstash-forwarder").size

    # Close and delete one of the files
    f1.close

    # Wait for prospector to realise it is deleted
    sleep 15

    # Check new size of registrar state
    File::Stat.new(".logstash-forwarder").size.should < s
  end
end
