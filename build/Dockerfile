FROM alpine:3.8

ENV OPERATOR=/usr/local/bin/keycloak-operator \
    USER_UID=1001 \
    USER_NAME=keycloak-operator

# install operator binary
COPY keycloak-operator ${OPERATOR}

COPY bin /usr/local/bin
COPY configs /usr/local/configs

RUN  chmod u+x /usr/local/bin/user_setup && chmod ugo+x /usr/local/bin/entrypoint && /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
