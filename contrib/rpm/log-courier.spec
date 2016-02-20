# Debug packaging does not work due to the go build building in extra debug sections RPM does not understand
# Maybe we patch something later to fix this, but for now just don't build a debug package
%define debug_package %{nil}

Summary: Log Courier
Name: log-courier
Version: 2.0
Release: 1%{dist}
License: Apache
Group: System Environment/Libraries
Packager: Jason Woods <packages@jasonwoods.me.uk>
URL: https://github.com/driskell/log-courier
Source: https://github.com/driskell/log-courier/archive/v%{version}.zip
BuildRoot: %{_tmppath}/%{name}-%{version}-root

BuildRequires: golang >= 1.5
BuildRequires: git

%if 0%{?rhel} >= 7
Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd
BuildRequires: systemd
%endif

Requires: zeromq3
Requires: logrotate

%description
Log Courier is a lightweight tool created to ship log files speedily and
securely, with low resource usage, to remote Logstash instances.

%prep
%setup -q -n %{name}-%{version}

%build
# Configure platform specific defaults
export LC_DEFAULT_CONFIGURATION_FILE=%{_sysconfdir}/log-courier/log-courier.yaml
export LC_DEFAULT_GENERAL_PERSIST_DIR=%{_var}/lib/log-courier
export LC_DEFAULT_GENERAL_ADMIN_BIND=unix:%{_var}/run/log-courier/admin.socket

# Enable vendor experiment in the event of Go 1.5 then generate and build
export GO15VENDOREXPERIMENT=1
go generate ./lc-lib/config ./lc-lib/core
go install . ./lc-admin ./lc-tlscert

%install
# Install binaries
mkdir -p %{buildroot}%{_sbindir}
install -m 0755 bin/log-courier %{buildroot}%{_sbindir}/log-courier
mkdir -p %{buildroot}%{_bindir}
install -m 0755 bin/lc-admin %{buildroot}%{_bindir}/lc-admin
install -m 0755 bin/lc-tlscert %{buildroot}%{_bindir}/lc-tlscert

# Install example configuration
mkdir -p %{buildroot}%{_sysconfdir}/log-courier %{buildroot}%{_sysconfdir}/log-courier/examples/
install -m 0644 docs/examples/* %{buildroot}%{_sysconfdir}/log-courier/examples/

# Make the run dir
mkdir -p %{buildroot}%{_var}/run %{buildroot}%{_var}/run/log-courier
touch %{buildroot}%{_var}/run/log-courier/admin.socket

# Install init script and related paraphernalia
%if 0%{?rhel} >= 7
mkdir -p %{buildroot}%{_unitdir}
# No systemd script in log-courier release yet
install -m 0644 contrib/initscripts/redhat-systemd.service %{buildroot}%{_unitdir}/log-courier.service
%else
mkdir -p %{buildroot}%{_sysconfdir}/init.d
install -m 0755 contrib/initscripts/redhat-sysv.init %{buildroot}%{_sysconfdir}/init.d/log-courier
touch %{buildroot}%{_var}/run/log-courier.pid
%endif
mkdir -p %{buildroot}%{_sysconfdir}/sysconfig
install -m 0644 contrib/initscripts/log-courier.env %{buildroot}%{_sysconfdir}/sysconfig/log-courier

# Make the state dir
mkdir -p %{buildroot}%{_var}/lib/log-courier
touch %{buildroot}%{_var}/lib/log-courier/.log-courier

%clean
rm -rf $RPM_BUILD_ROOT

%post
%if 0%{?rhel} >= 7
%systemd_post log-courier.service
%else
/sbin/chkconfig --add log-courier
%endif

%preun
%if 0%{?rhel} >= 7
%systemd_preun log-courier.service
%else
if [ $1 -eq 0 ]; then
	/sbin/service log-courier stop >/dev/null 2>&1
	/sbin/chkconfig --del log-courier
fi
%endif

%postun
%if 0%{?rhel} >= 7
%systemd_postun_with_restart log-courier.service
%else
if [ $1 -ge 1 ]; then
	/sbin/service log-courier restart >/dev/null 2>&1
fi
%endif

%files
%defattr(0755,root,root,0755)
%{_sbindir}/log-courier
%{_bindir}/lc-admin
%{_bindir}/lc-tlscert
%if 0%{?rhel} >= 7
%{_unitdir}/log-courier.service
%else
%{_sysconfdir}/init.d/log-courier
%endif

%defattr(0644,root,root,0755)
%dir %{_sysconfdir}/log-courier
%{_sysconfdir}/log-courier/examples
%config(noreplace) %{_sysconfdir}/sysconfig/log-courier
%if 0%{?rhel} < 7
%ghost %{_var}/run/log-courier.pid
%endif
%dir %attr(0700,root,root) %{_var}/run/log-courier
%ghost %{_var}/run/log-courier/admin.socket
%dir %{_var}/lib/log-courier
%ghost %{_var}/lib/log-courier/.log-courier

%changelog
* Thu Aug 6 2015 Jason Woods <devel@jasonwoods.me.uk> - 1.8-1
- Upgrade to v1.8

* Wed Jun 3 2015 Jason Woods <devel@jasonwoods.me.uk> - 1.7-1
- Upgrade to v1.7

* Sat Feb 28 2015 Jason Woods <devel@jasonwoods.me.uk> - 1.5-1
- Upgrade to v1.5

* Mon Jan 5 2015 Jason Woods <devel@jasonwoods.me.uk> - 1.3-1
- Upgrade to v1.3

* Wed Dec 3 2014 Jason Woods <devel@jasonwoods.me.uk> - 1.2-5
- Upgrade to v1.2 final

* Sat Nov 8 2014 Jason Woods <devel@jasonwoods.me.uk> - 1.2-4
- Upgrade to v1.2
- Fix stop message on future upgrade

* Wed Nov 5 2014 Jason Woods <devel@jasonwoods.me.uk> - 1.1-4
- Build with ZMQ 3 support

* Mon Nov 3 2014 Jason Woods <devel@jasonwoods.me.uk> - 1.1-3
- Fix init/systemd registration

* Sun Nov 2 2014 Jason Woods <devel@jasonwoods.me.uk> - 1.1-2
- Package for EL7
- Restart service on upgrade

* Fri Oct 31 2014 Jason Woods <devel@jasonwoods.me.uk> 1.1-1
- Released 1.1
- Cleanup for EL7 build

* Mon Oct 13 2014 Jason Woods <packages@jasonwoods.me.uk> 0.15.1-1
- Rebuild from v0.15 develop to fix more issues
- Label as v0.15.1

* Thu Sep 4 2014 Jason Woods <packages@jasonwoods.me.uk> 0.14.rc2-1
- Rebuild from develop to fix more issues and enable unix socket
	for administration
- Label as v0.14.rc2

* Wed Sep 3 2014 Jason Woods <packages@jasonwoods.me.uk> 0.14.rc1-1
- Rebuild from develop to fix various reconnect hang issues
- Label as v0.14.rc1

* Mon Sep 1 2014 Jason Woods <packages@jasonwoods.me.uk> 0.13-1
- Initial build of v0.13
