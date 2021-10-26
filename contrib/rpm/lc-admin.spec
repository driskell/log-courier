# Debug packaging does not work due to the go build building in extra debug sections RPM does not understand
# Maybe we patch something later to fix this, but for now just don't build a debug package
%define debug_package %{nil}

Summary: Administration Utility
Name: lc-admin
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

%description
lc-admin is a remote administration tool for the Log Courier Suite.

%prep
%setup -q -n log-courier-%{version}

%build
# Configure platform specific defaults
export LC_DEFAULT_CONFIGURATION_FILE=%{_sysconfdir}/log-courier/log-courier.yaml
export LC_DEFAULT_ADMIN_BIND=unix:%{_var}/run/log-courier/admin.socket
export LC_DEFAULT_CARVER_CONFIGURATION_FILE=%{_sysconfdir}/log-carver/log-carver.yaml
export LC_DEFAULT_CARVER_ADMIN_BIND=unix:%{_var}/run/log-carver/admin.socket

export GOBIN=%{_builddir}/bin
go generate -mod=vendor ./lc-admin
go install -mod=vendor ./lc-admin

%check
VERSION=$(%{_builddir}/bin/lc-admin --version)
VERSION=${VERSION#Admin version }
if [ ! -f .skip-version-check ] && [ "$VERSION" != "%{version}" ]; then
	exit 1
fi

%install
# Install binary
mkdir -p %{buildroot}%{_bindir}
install -m 0755 "%{_builddir}/bin/lc-admin" %{buildroot}%{_bindir}/lc-admin

# Install docs
mkdir -p %{buildroot}%{_docdir}/%{name}-%{version}
install -m 0644 docs/AdministrationUtility.md %{buildroot}%{_docdir}/%{name}-%{version}/

%clean
rm -rf $RPM_BUILD_ROOT

%files
%defattr(0755,root,root,0755)
%{_bindir}/lc-admin

%defattr(0644,root,root,0755)
%{_docdir}/%{name}-%{version}
