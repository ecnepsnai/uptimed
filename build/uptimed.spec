Name:           uptimed
Version:        ##VERSION##
Release:        1%{?dist}
Summary:        System uptime and reboot monitor

License:        Apache-2.0
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  systemd-rpm-macros

Provides:       %{name} = %{version}

%description
Uptimed is a simple golang application that can be used to monitor and alert on server reboots.

At a configured frequency is writes the current time to a specified file. When the application starts up it reads that
file and will post a discord notification saying that the server has booted and was last running at the date of the last
heartbeat.

%global debug_package %{nil}

%prep
%autosetup

%build
CGO_ENABLED=0 go build -ldflags="-s -w" -v -o %{name}

%install
install -Dpm 0755 %{name} %{buildroot}%{_bindir}/%{name}
install -Dpm 0755 uptimed.env %{buildroot}%{_sysconfdir}/uptimed.env
install -Dpm 644 %{name}.service %{buildroot}%{_unitdir}/%{name}.service

%check
CGO_ENABLED=0 go test -v

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%files
%{_sysconfdir}/uptimed.env
%{_bindir}/%{name}
%{_unitdir}/%{name}.service

