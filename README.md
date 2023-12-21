Проект по созданию оффлайн и онлайн нагрузки на системы Кредитного контроля

Оффлайн трафик формирует CDR файлы и передает их в каталог для дальнейшей обработки

Онлайн передается по Diameter + AVP или Camel(SCP) (через tvl decoder). При получении кода 4011 или Continum для Diameter формируется офлайн CDR

CAMEL работает только в режиме демона. При запуске CAMEL+BRT -> по приоритет отправки по CAMEL-BRT-OFFLINE


Use -d start deamon mode

Use -s stop deamon mode

Use -debug start with debug mode

Use -file save cdr to files(Offline)

Use -brt for test connection to BRT (Diameter)

Use -camel for UP SCP Server(Camel protocol)

Use -brtlist task list (local,roam)

Use -rm Delete all files in directories(Test optional)