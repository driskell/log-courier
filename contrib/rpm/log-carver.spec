# Debug packaging does not work due to the go build building in extra debug sections RPM does not understand
# Maybe we patch something later to fix this, but for now just don't build a debug package
%define debug_package %{nil}

Summary: Log Carver
Name: log-carver
Version: %%VERSION%%
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

Requires: lc-admin
Requires: lc-tlscert
Requires: logrotate
Requires: geolite2-city

%description
Log Carver is a lightweight tool created to process events from Log Courier speedily and securely, with low resource usage.

%prep
%setup -q -n log-courier-%{version}

%build
# Configure platform specific defaults
export LC_DEFAULT_CONFIGURATION_FILE=%{_sysconfdir}/log-carver/log-carver.yaml
export LC_DEFAULT_GEO_IP_ACTION_DATABASE=/usr/share/GeoIP/GeoLite2-City.mmdb
export LC_DEFAULT_ADMIN_BIND=unix:%{_var}/run/log-carver/admin.socket

export GOBIN=%{_builddir}/bin
go generate -mod=vendor ./log-carver ./lc-admin
go install -mod=vendor ./log-carver ./lc-admin

%check
VERSION=$(%{_builddir}/bin/log-carver --version)
VERSION=${VERSION#Log Carver version }
if [ ! -f .skip-version-check ] && [ "$VERSION" != "%{version}" ]; then
	exit 1
fi

%install
# Install binaries
mkdir -p %{buildroot}%{_sbindir}
install -m 0755 %{_builddir}/bin/log-carver %{buildroot}%{_sbindir}/log-carver

# Install config directory
mkdir -p %{buildroot}%{_sysconfdir}/log-carver

# Make the state dir
mkdir -p %{buildroot}%{_var}/lib/log-carver

# Install init script and related paraphernalia
mkdir -p %{buildroot}%{_sysconfdir}/sysconfig

%if 0%{?rhel} >= 7
mkdir -p %{buildroot}%{_unitdir}
install -m 0644 contrib/initscripts/log-carver-redhat-systemd.service %{buildroot}%{_unitdir}/log-carver.service
install -m 0644 contrib/initscripts/log-carver-systemd.env %{buildroot}%{_sysconfdir}/sysconfig/log-carver
%else
mkdir -p %{buildroot}%{_sysconfdir}/init.d
install -m 0755 contrib/initscripts/log-carver-redhat-sysv.init %{buildroot}%{_sysconfdir}/init.d/log-carver
install -m 0644 contrib/initscripts/log-carver.env %{buildroot}%{_sysconfdir}/sysconfig/log-carver
# Make the run dir
mkdir -p %{buildroot}%{_var}/run/log-carver
touch %{buildroot}%{_var}/run/log-carver/log-carver.pid
%endif

# Install docs
mkdir -p %{buildroot}%{_docdir}/%{name}-%{version}
install -m 0644 docs/log-carver/*.md %{buildroot}%{_docdir}/%{name}-%{version}/
mkdir -p %{buildroot}%{_docdir}/%{name}-%{version}/actions
install -m 0644 docs/log-carver/actions/*.md %{buildroot}%{_docdir}/%{name}-%{version}/actions/
mkdir -p %{buildroot}%{_docdir}/%{name}-%{version}/examples
install -m 0644 docs/log-carver/examples/*.yaml %{buildroot}%{_docdir}/%{name}-%{version}/examples/

%pre
if ! getent group log-carver >/dev/null; then
	groupadd log-carver
fi
if ! getent passwd log-carver >/dev/null; then
	useradd -r -d /var/lib/log-carver -s /sbin/nologin -g log-carver log-carver
fi

%clean
rm -rf $RPM_BUILD_ROOT

%post
%if 0%{?rhel} >= 7
%systemd_post log-carver.service
%else
/sbin/chkconfig --add log-carver
%endif

%preun
%if 0%{?rhel} >= 7
%systemd_preun log-carver.service
%else
if [ $1 -eq 0 ]; then
	/sbin/service log-carver stop >/dev/null 2>&1
	/sbin/chkconfig --del log-carver
fi
%endif

%postun
%if 0%{?rhel} >= 7
%systemd_postun_with_restart log-carver.service
%else
if [ $1 -ge 1 ]; then
	if [ -f /var/run/log-carver.pid ]; then
		mv /var/run/log-carver.pid /var/run/log-carver/log-carver.pid
	fi
	if /sbin/service log-carver status >/dev/null 2>&1; then
		/sbin/service log-carver restart >/dev/null 2>&1
	fi
fi
%endif

%files
%defattr(0755,root,root,0755)
%{_sbindir}/log-carver
%if 0%{?rhel} < 7
%{_sysconfdir}/init.d/log-carver
%endif

%defattr(0644,root,root,0755)
%if 0%{?rhel} >= 7
%{_unitdir}/log-carver.service
%endif
%dir %{_sysconfdir}/log-carver
%{_docdir}/%{name}-%{version}
%config(noreplace) %{_sysconfdir}/sysconfig/log-carver

%defattr(0644,log-carver,log-carver,0755)
%if 0%{?rhel} < 7
%ghost %{_var}/run/log-carver/log-carver.pid
%dir %attr(0700,log-carver,log-carver) %{_var}/run/log-carver
%ghost %{_var}/run/log-carver/admin.socket
%endif
%dir %{_var}/lib/log-carver
