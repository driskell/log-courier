# Debug packaging does not work due to the go build building in extra debug sections RPM does not understand
# Maybe we patch something later to fix this, but for now just don't build a debug package
%define debug_package %{nil}

Summary: SSL Certificate Utility
Name: lc-tlscert
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
lc-tlscert is a utility for quickly generating self-signed certificates.

%prep
%setup -q -n %{name}-%{version}

%build
export GOBIN=%{_builddir}/bin
go generate -mod=vendor ./lc-tlscert
go install -mod=vendor ./lc-tlscert

%install
# Install binary
mkdir -p %{buildroot}%{_bindir}
install -m 0755 "%{_builddir}/bin/lc-tlscert" %{buildroot}%{_bindir}/lc-tlscert

# Install docs
mkdir -p %{buildroot}%{_docdir}/%{name}-%{version}
install -m 0644 docs/log-courier/SSLCertificateUtility.md %{buildroot}%{_docdir}/%{name}-%{version}/

%clean
rm -rf $RPM_BUILD_ROOT

%files
%defattr(0755,root,root,0755)
%{_bindir}/lc-tlscert
