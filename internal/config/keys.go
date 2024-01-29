package config

//Keys that are used in the config file

const CReverseProxyHost = "reverseproxy.host"
const CReverseProxyPort = "reverseproxy.port"
const CReverseProxySessionHeader = "reverseproxy.session.header"
const CReverseProxySessionSalt = "reverseproxy.session.salt"
const CReverseProxySessionTimeout = "reverseproxy.session.timeout" //Timeout in seconds
const CReverseProxyPool = "reverseproxy.pool"                      //Basic number of containers that will be created

const CMgmtHost = "mgmt.host"
const CMgmtPort = "mgmt.port"
const CMgmtKey = "mgmt.key" //Key used to authenticate to the management interface

const CDockerHost = "docker.host"

// Network used by the reverse proxy. This network will be injected into the main container
const CDockerContainerName = "docker.container-name" //Name of the container that will be created
const CDockerComposeWorkdir = "docker.compose.workdir"
const CDockerComposeFile = "docker.compose.file" //File of the docker compose file
