# Debug packaging does not work due to the go build building in extra debug sections RPM does not understand
# Maybe we patch something later to fix this, but for now just don't build a debug package
%define debug_package %{nil}

Summary: Log Courier
Name: log-courier
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

Requires: logrotate

%description
Log Courier is a lightweight tool created to ship log files speedily and securely, with low resource usage, to remote Logstash or Log Carver instances.

%prep
%setup -q -n %{name}-%{version}

%build
# Configure platform specific defaults
export LC_DEFAULT_CONFIGURATION_FILE=%{_sysconfdir}/log-courier/log-courier.yaml
export LC_DEFAULT_GENERAL_PERSIST_DIR=%{_var}/lib/log-courier
export LC_DEFAULT_ADMIN_BIND=unix:%{_var}/run/log-courier/admin.socket

export GOBIN=%{_builddir}/bin
go generate -mod=vendor . ./lc-admin ./lc-tlscert
go install -mod=vendor . ./lc-admin ./lc-tlscert

%check
VERSION=$(%{_builddir}/bin/log-courier --version)
VERSION=${VERSION#Log Courier version }
if [ "$VERSION" != "%{version}" ]; then
	exit 1
fi

%install
# Install binaries
mkdir -p %{buildroot}%{_sbindir}
install -m 0755 %{_builddir}/bin/log-courier %{buildroot}%{_sbindir}/log-courier
mkdir -p %{buildroot}%{_bindir}
install -m 0755 "%{_builddir}/bin/lc-admin" %{buildroot}%{_bindir}/lc-admin
install -m 0755 "%{_builddir}/bin/lc-tlscert" %{buildroot}%{_bindir}/lc-tlscert

# Install config directory
mkdir -p %{buildroot}%{_sysconfdir}/log-carver

# Make the state dir
mkdir -p %{buildroot}%{_var}/lib/log-courier
touch %{buildroot}%{_var}/lib/log-courier/.log-courier

# Install init script and related paraphernalia
mkdir -p %{buildroot}%{_sysconfdir}/sysconfig

%if 0%{?rhel} >= 7
mkdir -p %{buildroot}%{_unitdir}
install -m 0644 contrib/initscripts/redhat-systemd.service %{buildroot}%{_unitdir}/log-courier.service
install -m 0644 contrib/initscripts/log-courier-systemd.env %{buildroot}%{_sysconfdir}/sysconfig/log-courier
%else
mkdir -p %{buildroot}%{_sysconfdir}/init.d
install -m 0755 contrib/initscripts/redhat-sysv.init %{buildroot}%{_sysconfdir}/init.d/log-courier
install -m 0644 contrib/initscripts/log-courier.env %{buildroot}%{_sysconfdir}/sysconfig/log-courier
# Make the run dir
mkdir -p %{buildroot}%{_var}/run/log-courier
touch %{buildroot}%{_var}/run/log-courier/admin.socket
touch %{buildroot}%{_var}/run/log-courier/log-courier.pid
%endif

# Install docs
mkdir -p %{buildroot}%{_docdir}/%{name}-%{version}
install -m 0644 docs/log-courier/*.md %{buildroot}%{_docdir}/%{name}-%{version}/
mkdir -p %{buildroot}%{_docdir}/%{name}-%{version}/codecs
install -m 0644 docs/log-courier/codecs/*.md %{buildroot}%{_docdir}/%{name}-%{version}/codecs/
mkdir -p %{buildroot}%{_docdir}/%{name}-%{version}/examples
install -m 0644 docs/log-courier/examples/*.conf %{buildroot}%{_docdir}/%{name}-%{version}/examples/
install -m 0644 docs/log-courier/examples/*.yaml %{buildroot}%{_docdir}/%{name}-%{version}/examples/

%pre
if ! getent group log-courier >/dev/null; then
	groupadd log-courier
fi
if ! getent passwd log-courier >/dev/null; then
	useradd -r -d /var/lib/log-courier -s /sbin/nologin -g log-courier log-courier
fi

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
	if [ -f /var/run/log-courier.pid ]; then
		mv /var/run/log-courier.pid /var/run/log-courier/log-courier.pid
	fi
	if /sbin/service log-courier status >/dev/null 2>&1; then
		/sbin/service log-courier restart >/dev/null 2>&1
	fi
fi
%endif

%files
%defattr(0755,root,root,0755)
%{_sbindir}/log-courier
%{_bindir}/lc-admin
%{_bindir}/lc-tlscert
%if 0%{?rhel} < 7
%{_sysconfdir}/init.d/log-courier
%endif

%defattr(0644,root,root,0755)
%if 0%{?rhel} >= 7
%{_unitdir}/log-courier.service
%endif
%dir %{_sysconfdir}/log-courier
%{_docdir}/%{name}-%{version}
%config(noreplace) %{_sysconfdir}/sysconfig/log-courier

%defattr(0644,log-courier,log-courier,0755)
%if 0%{?rhel} < 7
%ghost %{_var}/run/log-courier/log-courier.pid
%dir %attr(0700,log-courier,log-courier) %{_var}/run/log-courier
%ghost %{_var}/run/log-courier/admin.socket
%endif
%dir %{_var}/lib/log-courier
%ghost %{_var}/lib/log-courier/.log-courier
