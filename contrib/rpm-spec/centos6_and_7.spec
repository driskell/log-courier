%if 0%{?rhel} > 6
%global _with_systemd --with-systemd
%endif
%global use_systemd %{!?_with_systemd:0}%{?_with_systemd:1}
%define  debug_package %{nil}

Summary:	Logstash logs shipper
Name:		log-courier
Version:	1.1
Release:	1
License:	GPL
Group:		System Environment/Daemons
URL:		https://github.com/driskell/log-courier
Source0:	https://github.com/driskell/log-courier/archive/log-courier-%{version}.tar.gz
Source1:	log-courier.systemd
Source2:	log-courier.sysv-init
Source3:	log-courier.conf
Source4:	log-courier.httpd.conf.example
Source5:	log-courier.sysconfig
BuildRoot:	%{_tmppath}/%{name}-%{version}-%{release}-root-%(id -nu)
BuildRequires:	golang, git
%if %{use_systemd}
Requires(pre):	systemd
Requires(post):	systemd
Requires(preun): systemd
Requires(postun): systemd
%else
Requires(post):	/sbin/chkconfig
Requires(preun): /sbin/chkconfig
%endif

%description
Log Courier is a tool created to ship log files speedily and securely to 
remote Logstash instances for processing whilst using small amounts of local resources.

%prep
%setup -q
cp -p %{SOURCE1} %{SOURCE2} %{SOURCE3} %{SOURCE4} %{SOURCE5} ./

%build
make

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}{%{_sbindir},%{_initrddir}}
mkdir -p %{buildroot}%{_docdir}/log-courier/examples
mkdir -p %{buildroot}%{_prefix}/lib/systemd/system/
mkdir -p %{buildroot}%{_sysconfdir}/log-courier   
mkdir -p %{buildroot}%{_sysconfdir}/log-courier/conf.d
mkdir -p %{buildroot}%{_sysconfdir}/sysconfig
mkdir -p %{buildroot}%{_sharedstatedir}/log-courier

install -p -m 755 bin/lc-admin			%{buildroot}%{_sbindir}/
install -p -m 755 bin/lc-tlscert		%{buildroot}%{_sbindir}/
install -p -m 755 bin/log-courier		%{buildroot}%{_sbindir}/

install -p -m 644 docs/examples/example-single.conf %{buildroot}%{_sysconfdir}/log-courier/log-courier.conf

install -p -m 644 docs/examples/*					%{buildroot}%{_docdir}/log-courier/examples
install -p -m 644 README.md							%{buildroot}%{_docdir}/log-courier
install -p -m 644 log-courier.conf  				%{buildroot}%{_sysconfdir}/log-courier
install -p -m 644 log-courier.httpd.conf.example 	%{buildroot}%{_sysconfdir}/log-courier/conf.d

install -p -m 644 log-courier.sysconfig %{buildroot}%{_sysconfdir}/sysconfig/log-courier
%if %{use_systemd}
install -p -m 755 log-courier.systemd	%{buildroot}/usr/lib/systemd/system/log-courier.service
%else
install -p -m 755 log-courier.sysv-init	%{buildroot}%{_initrddir}/log-courier
%endif


%clean
rm -rf %{buildroot}

%pre

%post
%if %{use_systemd}
	/bin/systemctl daemon-reload &>/dev/null || :
%else
if [ $1 -eq 1 ]; then
	# Initial installation
 	/sbin/chkconfig --add log-courier || :
fi
%endif

%preun
if [ $1 -eq 0 ]; then
# 	# Package removal, not upgrade
%if %{use_systemd}
	%systemd_preun log-courier.service
%else
	%{_initrddir}/log-courier stop &>/dev/null || :
	/sbin/chkconfig --del log-courier || :
%endif
fi

%postun
%if %{use_systemd}
	/bin/systemctl daemon-reload &>/dev/null || :
%endif
if [ $1 -ge 1 ]; then
	# Package upgrade, not uninstall
%if %{use_systemd}
	%systemd_postun_with_restart log-courier.service
%else
	%{_initrddir}/log-courier condrestart &>/dev/null || :
%endif
fi

%files
%{_sbindir}/lc-admin
%{_sbindir}/lc-tlscert
%{_sbindir}/log-courier
%doc %{_docdir}/log-courier/*
%dir %{_sharedstatedir}/log-courier
%config(noreplace) %{_sysconfdir}/log-courier/log-courier.conf
%config(noreplace) %{_sysconfdir}/log-courier/conf.d/log-courier.httpd.conf.example
%config(noreplace) %{_sysconfdir}/sysconfig/log-courier
%if %{use_systemd}
	/usr/lib/systemd/system/log-courier.service
%else
	%{_initrddir}/log-courier
%endif

