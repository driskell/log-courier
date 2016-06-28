# Debug packaging does not work due to the go build building in extra debug sections RPM does not understand
# Maybe we patch something later to fix this, but for now just don't build a debug package
%define debug_package %{nil}

Summary: Fact Courier
Name: fact-courier
Version: 2.5.0
Release: 1%{dist}
License: Apache
Group: System Environment/Libraries
Packager: Jason Woods <packages@jasonwoods.me.uk>
URL: https://github.com/driskell/log-courier
Source: https://github.com/driskell/log-courier/archive/v%{version}.zip
BuildRoot: %{_tmppath}/%{name}-%{version}-root

# Get this from the great Jason Brooks:
#   https://copr.fedorainfracloud.org/coprs/jasonbrooks/docker/package/golang/
# We could also get from a RedHat dev, but currently broken and leaking out 1.6:
#   https://copr.fedorainfracloud.org/coprs/jcajka/golang1.5/package/golang/
# For RHEL6, 1.5.1 can be obtained from EPEL, but it is not available to RHEL7
BuildRequires: golang >= 1.5
BuildRequires: git

%if 0%{?rhel} >= 7
Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd
BuildRequires: systemd
%endif

Requires: logrotate
Requires: munin-node

%description
Fact Courier is a lightweight tool created to ship the results of munin-node
scripts speedily and securely, with low resource usage, to remote Logstash
instances. The munin-node service does not even need to be running - the scripts
are called directly.

%prep
%setup -q -n %{name}-%{version}

%build
# Build a go workspace
mkdir -p _workspace/src/github.com/driskell
ln -nsf $(pwd) _workspace/src/github.com/driskell/log-courier
export GOPATH=$(pwd)/_workspace
cd "$GOPATH/src/github.com/driskell/log-courier"

# Configure platform specific defaults
export LC_FACT_DEFAULT_CONFIGURATION_FILE=%{_sysconfdir}/fact-courier/fact-courier.yaml

# Enable vendor experiment in the event of Go 1.5 then generate and build
export GO15VENDOREXPERIMENT=1
go generate .
go install ./fact-courier

%install
export GOPATH=$(pwd)/_workspace

# Install binaries
mkdir -p %{buildroot}%{_sbindir}
install -m 0755 $GOPATH/bin/fact-courier %{buildroot}%{_sbindir}/fact-courier

# Install init script and related paraphernalia
mkdir -p %{buildroot}%{_sysconfdir}/sysconfig
%if 0%{?rhel} >= 7
mkdir -p %{buildroot}%{_unitdir}
install -m 0644 contrib/initscripts/fact-redhat-systemd.service %{buildroot}%{_unitdir}/fact-courier.service
install -m 0644 contrib/initscripts/fact-courier-systemd.env %{buildroot}%{_sysconfdir}/sysconfig/fact-courier
%else
# Make the run dir
mkdir -p %{buildroot}%{_var}/run %{buildroot}%{_var}/run/fact-courier

mkdir -p %{buildroot}%{_sysconfdir}/init.d
install -m 0755 contrib/initscripts/fact-redhat-sysv.init %{buildroot}%{_sysconfdir}/init.d/fact-courier
install -m 0644 contrib/initscripts/fact-courier.env %{buildroot}%{_sysconfdir}/sysconfig/fact-courier
touch %{buildroot}%{_var}/run/fact-courier/fact-courier.pid
%endif

%pre
if ! getent group fact-courier >/dev/null; then
	groupadd fact-courier
fi
if ! getent passwd fact-courier >/dev/null; then
	useradd -r -d /var/lib/fact-courier -s /sbin/nologin -g fact-courier fact-courier
fi

%clean
rm -rf $RPM_BUILD_ROOT

%post
%if 0%{?rhel} >= 7
%systemd_post fact-courier.service
%else
/sbin/chkconfig --add fact-courier
%endif

%preun
%if 0%{?rhel} >= 7
%systemd_preun fact-courier.service
%else
if [ $1 -eq 0 ]; then
	/sbin/service fact-courier stop >/dev/null 2>&1
	/sbin/chkconfig --del fact-courier
fi
%endif

%postun
%if 0%{?rhel} >= 7
%systemd_postun_with_restart fact-courier.service
%else
if [ $1 -ge 1 ]; then
	if [ -f /var/run/fact-courier.pid ]; then
		mv /var/run/fact-courier.pid /var/run/fact-courier/fact-courier.pid
	fi
	if /sbin/service fact-courier status >/dev/null 2>&1; then
		/sbin/service fact-courier restart >/dev/null 2>&1
	fi
fi
%endif

%files
%defattr(0755,root,root,0755)
%{_sbindir}/fact-courier
%if 0%{?rhel} >= 7
%{_unitdir}/fact-courier.service
%else
%{_sysconfdir}/init.d/fact-courier
%endif

%defattr(0644,root,root,0755)
%dir %{_sysconfdir}/fact-courier
%config(noreplace) %{_sysconfdir}/sysconfig/fact-courier

%defattr(0644,fact-courier,fact-courier,0755)
%if 0%{?rhel} < 7
%ghost %{_var}/run/fact-courier/fact-courier.pid
%dir %attr(0700,fact-courier,fact-courier) %{_var}/run/fact-courier
%endif

%changelog
* Tue Jun 28 2016 Jason Woods <devel@jasonwoods.me.uk> - 2.5.0-1
- Fact Courier
