# Copyright 2014-2021 Jason Woods.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

require 'cabin'
require 'log-courier/server'

# Common helpers for testing both ruby client and the courier
shared_context 'Helpers' do
  before :all do
    Thread.abort_on_exception = true

    @ssl_cert = File.open(File.join(TEMP_PATH, 'ssl_cert'), 'w')
    @ssl_key = File.open(File.join(TEMP_PATH, 'ssl_key'), 'w')
    @ssl_csr = File.open(File.join(TEMP_PATH, 'ssl_csr'), 'w')

    # Generate the ssl key
    system("openssl req -config spec/lib/openssl.cnf -new -batch -keyout #{@ssl_key.path} -out #{@ssl_csr.path}")
    system(
      "openssl x509 -extfile spec/lib/openssl.cnf -extensions extensions_section -req -days 365 -in #{@ssl_csr.path}" \
      " -signkey #{@ssl_key.path} -out #{@ssl_cert.path}",
    )
  end

  after :all do
    [@ssl_cert, @ssl_key, @ssl_csr].each do |f|
      File.unlink(f.path) if File.file?(f.path)
    end
  end

  before :each do
    # When we add a file we log it here, so after we can remove them
    @files = []

    @event_queue = SizedQueue.new 10_000

    @servers = {}
    @server_counts = {}
    @server_threads = {}
  end

  after :each do
    # Remove any files we created for the test
    @files.each(&:close)

    @files = []
  end

  # A helper that starts a Log Courier server
  def start_server(**args)
    args = {
      id: '__default__',
      transport: 'tls',
    }.merge!(**args)

    id = args[:id]

    logger = Cabin::Channel.new
    logger.subscribe $stdout
    logger['instance'] = id
    logger.level = :debug

    raise 'Server already initialised' if @servers.key?(id)

    # Reset server for each test
    @servers[id] = LogCourier::Server.new(
      ssl_certificate: @ssl_cert.path,
      ssl_key: @ssl_key.path,
      logger: logger,
      **args,
    )

    @server_counts[id] = 0
    @server_threads[id] = Thread.new do
      @servers[id].run do |event|
        @server_counts[id] += 1
        @event_queue << event
      end
    rescue LogCourier::ShutdownSignal
      0
    end
  end

  # A helper to shutdown a Log Courier server
  def shutdown_server(which = nil)
    which = if which.nil?
              @servers.keys
            else
              [which]
            end
    which.each do |id|
      @server_threads[id].raise LogCourier::ShutdownSignal
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
    path ||= File.join(TEMP_PATH, 'logs', "log-#{@files.length}")

    # Return a new file for testing, and log it for cleanup afterwards
    f = type.new(path)
    @files.push(f)
    f
  end

  # Rename a log file and create a new one in its place
  def rotate(logfile, prefix = '')
    old_name = logfile.path

    new_name = if prefix == ''
                 "#{logfile.path}r"
               else
                 File.join(File.dirname(f.path), "#{prefix}#{File.basename(f.path)}r")
               end

    logfile.rename new_name

    create_log(logfile.class, old_name)
  end

  def receive_and_check(args = {}, &block)
    args = {
      total: nil,
      check: true,
      check_file: true,
      check_order: true,
      host: nil,
    }.merge!(args)

    # Quick check of the total events we are expecting - but allow time to receive them
    total = if args[:total].nil?
              @files.reduce(0) do |sum, f|
                sum + f.count
              end
            else
              args[:total]
            end

    args.delete_if do |_, v|
      v.nil?
    end

    orig_total = total
    check = args[:check]

    waited = 0
    while total.positive? && waited <= EVENT_WAIT_COUNT
      if @event_queue.length.zero?
        sleep(EVENT_WAIT_TIME)
        waited += 1
        next
      end

      waited = 0
      until @event_queue.length.zero?
        e = @event_queue.pop
        total -= 1
        next unless check

        if block.nil?
          found = @files.find do |f|
            next unless f.pending?

            f.logged?({ event: e }.merge!(args))
          end
          expect(found).to_not be_nil, "Event received not recognised: #{e}"
        else
          block.call e
        end
      end
    end

    # Fancy calculation to give a nice "expected" output of expected num of events
    expect(orig_total - total).to eq orig_total
  end
end
