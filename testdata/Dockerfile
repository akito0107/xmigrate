FROM postgres:9.6.10
RUN localedef -i ja_JP -c -f UTF-8 -A /usr/share/locale/locale.alias ja_JP.UTF-8
ENV LANG ja_JP.UTF-8

COPY sql/*.sql /docker-entrypoint-initdb.d/

EXPOSE 5432
