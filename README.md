Проект по созданию оффлайн и онлайн нагрузки на системы Кредитного контроля

Оффлайн трафик формирует CDR файлы и передает их в каталог для дальнейшей обработки

Онлайн передается по Diameter + AVP или Camel (через tvl decoder). При получении кода 4011 или Continum формируется офлайн CDR


Use -d start deamon mode

Use -s stop deamon mode

Use -debug start with debug mode

Use -file save cdr to files(Offline)

Use -brt for test connection to BRT (Diameter)

Use -camel for test connection to BRT (Camel)

Use -brtlist task list (local,roam)