Проект по созданию оффлайн и онлайн нагрузки на системы Кредитного контроля

Оффлайн трафик формирует CDR файлы и передает их в каталог для дальнейшей обработки

Онлайн передается по Diameter + AVP или Camel(SCP) (через tvl decoder). При получении кода 4011 или Continum для Diameter формируется офлайн CDR

CAMEL работает только в режиме демона. При запуске CAMEL+BRT -> по приоритет отправки по CAMEL-BRT-OFFLINE


Project to create offline and online load on Credit control systems(3GPP)

Offline traffic generates CDR files and transfers them to the directory for further processing

Online is transmitted via Diameter + AVP or Camel(SCP) (via tvl decoder). When receiving code 4011 or Continum for Diameter, an offline CDR is generated

CAMEL only works in daemon mode. When running CAMEL+BRT -> by priority sending via CAMEL-BRT-OFFLINE

# Parameters 
Use -d start deamon mode

Use -s stop deamon mode

Use -debug start with debug mode

Use -file save cdr to files(Offline)

Use -brt for test connection to BRT (Diameter)

Use -brtlist task list (local,roam)

Use -camel for UP SCP Server(Camel protocol)

# Test parameters
Use -rm Delete all files in directories(Test optional)

Use -slow1 to send 1 message every 10 seconds

