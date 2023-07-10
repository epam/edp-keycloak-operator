# API Reference

Packages:

- [v1.edp.epam.com/v1alpha1](#v1edpepamcomv1alpha1)
- [v1.edp.epam.com/v1](#v1edpepamcomv1)

# v1.edp.epam.com/v1alpha1

Resource Types:

- [ClusterKeycloak](#clusterkeycloak)

- [KeycloakAuthFlow](#keycloakauthflow)

- [KeycloakClient](#keycloakclient)

- [KeycloakClientScope](#keycloakclientscope)

- [KeycloakRealmComponent](#keycloakrealmcomponent)

- [KeycloakRealmGroup](#keycloakrealmgroup)

- [KeycloakRealmIdentityProvider](#keycloakrealmidentityprovider)

- [KeycloakRealmRoleBatch](#keycloakrealmrolebatch)

- [KeycloakRealmRole](#keycloakrealmrole)

- [KeycloakRealm](#keycloakrealm)

- [KeycloakRealmUser](#keycloakrealmuser)

- [Keycloak](#keycloak)




## ClusterKeycloak
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>






ClusterKeycloak is the Schema for the clusterkeycloaks API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>ClusterKeycloak</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakspec">spec</a></b></td>
        <td>object</td>
        <td>
          ClusterKeycloakSpec defines the desired state of ClusterKeycloak.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakstatus">status</a></b></td>
        <td>object</td>
        <td>
          ClusterKeycloakStatus defines the observed state of ClusterKeycloak.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloak.spec
<sup><sup>[↩ Parent](#clusterkeycloak)</sup></sup>



ClusterKeycloakSpec defines the desired state of ClusterKeycloak.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>secret</b></td>
        <td>string</td>
        <td>
          Secret is a secret name which contains admin credentials.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          URL of keycloak service.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>adminType</b></td>
        <td>enum</td>
        <td>
          AdminType can be user or serviceAccount, if serviceAccount was specified, then client_credentials grant type should be used for getting admin realm token.<br/>
          <br/>
            <i>Enum</i>: serviceAccount, user<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloak.status
<sup><sup>[↩ Parent](#clusterkeycloak)</sup></sup>



ClusterKeycloakStatus defines the observed state of ClusterKeycloak.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>connected</b></td>
        <td>boolean</td>
        <td>
          Connected shows if keycloak service is up and running.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>

## KeycloakAuthFlow
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>








<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakAuthFlow</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakauthflowspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakauthflowstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakAuthFlow.spec
<sup><sup>[↩ Parent](#keycloakauthflow-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          Alias is display name for authentication flow<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>builtIn</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>providerId</b></td>
        <td>string</td>
        <td>
          ProviderID for root auth flow and provider for child auth flows<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Realm is name of keycloak realm<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>topLevel</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakauthflowspecauthenticationexecutionsindex-1">authenticationExecutions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>childType</b></td>
        <td>string</td>
        <td>
          ChildType is type for auth flow if it has a parent, available options: basic-flow, form-flow<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>parentName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakAuthFlow.spec.authenticationExecutions[index]
<sup><sup>[↩ Parent](#keycloakauthflowspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>authenticator</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakauthflowspecauthenticationexecutionsindexauthenticatorconfig-1">authenticatorConfig</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>authenticatorFlow</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>priority</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requirement</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakAuthFlow.spec.authenticationExecutions[index].authenticatorConfig
<sup><sup>[↩ Parent](#keycloakauthflowspecauthenticationexecutionsindex-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakAuthFlow.status
<sup><sup>[↩ Parent](#keycloakauthflow-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakClient
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>






KeycloakClient is the Schema for the keycloakclients API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakClient</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakClientSpec defines the desired state of KeycloakClient.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakClientStatus defines the observed state of KeycloakClient.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec
<sup><sup>[↩ Parent](#keycloakclient-1)</sup></sup>



KeycloakClientSpec defines the desired state of KeycloakClient.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientId</b></td>
        <td>string</td>
        <td>
          ClientId is a unique keycloak client ID referenced in URI and tokens.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>advancedProtocolMappers</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clientRoles</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>defaultClientScopes</b></td>
        <td>[]string</td>
        <td>
          A list of default client scopes for a keycloak client.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>directAccess</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>frontChannelLogout</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecprotocolmappersindex-1">protocolMappers</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>public</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecrealmrolesindex-1">realmRoles</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reconciliationStrategy</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: full, addOnly<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secret</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecserviceaccount-1">serviceAccount</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetRealm</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>webUrl</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.protocolMappers[index]
<sup><sup>[↩ Parent](#keycloakclientspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocolMapper</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.realmRoles[index]
<sup><sup>[↩ Parent](#keycloakclientspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>composite</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.serviceAccount
<sup><sup>[↩ Parent](#keycloakclientspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>attributes</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecserviceaccountclientrolesindex-1">clientRoles</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>realmRoles</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.serviceAccount.clientRoles[index]
<sup><sup>[↩ Parent](#keycloakclientspecserviceaccount-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>roles</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.status
<sup><sup>[↩ Parent](#keycloakclient-1)</sup></sup>



KeycloakClientStatus defines the observed state of KeycloakClient.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clientSecretName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakClientScope
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>








<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakClientScope</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakclientscopespec-1">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientscopestatus-1">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClientScope.spec
<sup><sup>[↩ Parent](#keycloakclientscope-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of keycloak client scope<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          Protocol is SSO protocol configuration which is being supplied by this client scope<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Realm is name of keycloak realm<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>default</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientscopespecprotocolmappersindex-1">protocolMappers</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClientScope.spec.protocolMappers[index]
<sup><sup>[↩ Parent](#keycloakclientscopespec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocolMapper</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClientScope.status
<sup><sup>[↩ Parent](#keycloakclientscope-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmComponent
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>








<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmComponent</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmcomponentspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmcomponentstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmComponent.spec
<sup><sup>[↩ Parent](#keycloakrealmcomponent-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>providerId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>providerType</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>config</b></td>
        <td>map[string][]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmComponent.status
<sup><sup>[↩ Parent](#keycloakrealmcomponent-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmGroup
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>








<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmGroup</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmgroupspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmgroupstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmGroup.spec
<sup><sup>[↩ Parent](#keycloakrealmgroup-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>access</b></td>
        <td>map[string]boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string][]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmgroupspecclientrolesindex-1">clientRoles</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>realmRoles</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subGroups</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmGroup.spec.clientRoles[index]
<sup><sup>[↩ Parent](#keycloakrealmgroupspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>roles</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmGroup.status
<sup><sup>[↩ Parent](#keycloakrealmgroup-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmIdentityProvider
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>








<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmIdentityProvider</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmidentityproviderspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmidentityproviderstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmIdentityProvider.spec
<sup><sup>[↩ Parent](#keycloakrealmidentityprovider-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>providerId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>addReadTokenRoleOnCreate</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>authenticateByDefault</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>displayName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>firstBrokerLoginFlowAlias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>linkOnly</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmidentityproviderspecmappersindex-1">mappers</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storeToken</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>trustEmail</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmIdentityProvider.spec.mappers[index]
<sup><sup>[↩ Parent](#keycloakrealmidentityproviderspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>identityProviderAlias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>identityProviderMapper</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmIdentityProvider.status
<sup><sup>[↩ Parent](#keycloakrealmidentityprovider-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmRoleBatch
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>








<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmRoleBatch</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolebatchspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolebatchstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmRoleBatch.spec
<sup><sup>[↩ Parent](#keycloakrealmrolebatch-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolebatchspecrolesindex-1">roles</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealmRoleBatch.spec.roles[index]
<sup><sup>[↩ Parent](#keycloakrealmrolebatchspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string][]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>composite</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolebatchspecrolesindexcompositesindex-1">composites</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>isDefault</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmRoleBatch.spec.roles[index].composites[index]
<sup><sup>[↩ Parent](#keycloakrealmrolebatchspecrolesindex-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealmRoleBatch.status
<sup><sup>[↩ Parent](#keycloakrealmrolebatch-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmRole
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>








<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmRole</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolespec-1">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolestatus-1">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmRole.spec
<sup><sup>[↩ Parent](#keycloakrealmrole-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string][]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>composite</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolespeccompositesindex-1">composites</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>isDefault</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmRole.spec.composites[index]
<sup><sup>[↩ Parent](#keycloakrealmrolespec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealmRole.status
<sup><sup>[↩ Parent](#keycloakrealmrole-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealm
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>






KeycloakRealm is the Schema for the keycloakrealms API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealm</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmSpec defines the desired state of KeycloakRealm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmStatus defines the observed state of KeycloakRealm.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec
<sup><sup>[↩ Parent](#keycloakrealm-1)</sup></sup>



KeycloakRealmSpec defines the desired state of KeycloakRealm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>realmName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>browserFlow</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>browserSecurityHeaders</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>disableCentralIDPMappers</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>frontendUrl</b></td>
        <td>string</td>
        <td>
          FrontendURL Set the frontend URL for the realm. Use in combination with the default hostname provider to override the base URL for frontend requests for a specific realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>keycloakOwner</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecpasswordpolicyindex-1">passwordPolicy</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecrealmeventconfig-1">realmEventConfig</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ssoAutoRedirectEnabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ssoRealmEnabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecssorealmmappersindex-1">ssoRealmMappers</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ssoRealmName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecthemes-1">themes</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecusersindex-1">users</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.passwordPolicy[index]
<sup><sup>[↩ Parent](#keycloakrealmspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.realmEventConfig
<sup><sup>[↩ Parent](#keycloakrealmspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>adminEventsDetailsEnabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>adminEventsEnabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabledEventTypes</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>eventsEnabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>eventsExpiration</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>eventsListeners</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.ssoRealmMappers[index]
<sup><sup>[↩ Parent](#keycloakrealmspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>identityProviderMapper</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.themes
<sup><sup>[↩ Parent](#keycloakrealmspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accountTheme</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>adminConsoleTheme</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>emailTheme</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>internationalizationEnabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>loginTheme</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.users[index]
<sup><sup>[↩ Parent](#keycloakrealmspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>username</b></td>
        <td>string</td>
        <td>
          Username of keycloak user<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realmRoles</b></td>
        <td>[]string</td>
        <td>
          RealmRoles is a list of roles attached to keycloak user<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.status
<sup><sup>[↩ Parent](#keycloakrealm-1)</sup></sup>



KeycloakRealmStatus defines the observed state of KeycloakRealm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>available</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmUser
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>








<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmUser</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmuserspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmuserstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmUser.spec
<sup><sup>[↩ Parent](#keycloakrealmuser-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>username</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>email</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>emailVerified</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>firstName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groups</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>keepResource</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>lastName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>password</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reconciliationStrategy</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requiredUserActions</b></td>
        <td>[]string</td>
        <td>
          RequiredUserActions is required action when user log in, example: CONFIGURE_TOTP, UPDATE_PASSWORD, UPDATE_PROFILE, VERIFY_EMAIL<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>roles</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmUser.status
<sup><sup>[↩ Parent](#keycloakrealmuser-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Keycloak
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>






Keycloak is the Schema for the keycloaks API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Keycloak</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakSpec defines the desired state of Keycloak.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakStatus defines the observed state of Keycloak.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Keycloak.spec
<sup><sup>[↩ Parent](#keycloak-1)</sup></sup>



KeycloakSpec defines the desired state of Keycloak.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>secret</b></td>
        <td>string</td>
        <td>
          Secret is the name of the k8s object Secret related to keycloak<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          URL of keycloak service<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>adminType</b></td>
        <td>enum</td>
        <td>
          AdminType can be user or serviceAccount, if serviceAccount was specified, then client_credentials grant type should be used for getting admin realm token<br/>
          <br/>
            <i>Enum</i>: serviceAccount, user<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>installMainRealm</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>realmName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ssoRealmName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakspecusersindex">users</a></b></td>
        <td>[]object</td>
        <td>
          Users is a list of keycloak users<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Keycloak.spec.users[index]
<sup><sup>[↩ Parent](#keycloakspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>username</b></td>
        <td>string</td>
        <td>
          Username of keycloak user<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realmRoles</b></td>
        <td>[]string</td>
        <td>
          RealmRoles is a list of roles attached to keycloak user<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Keycloak.status
<sup><sup>[↩ Parent](#keycloak-1)</sup></sup>



KeycloakStatus defines the observed state of Keycloak.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>connected</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>

# v1.edp.epam.com/v1

Resource Types:

- [KeycloakAuthFlow](#keycloakauthflow)

- [KeycloakClient](#keycloakclient)

- [KeycloakClientScope](#keycloakclientscope)

- [KeycloakRealmComponent](#keycloakrealmcomponent)

- [KeycloakRealmGroup](#keycloakrealmgroup)

- [KeycloakRealmIdentityProvider](#keycloakrealmidentityprovider)

- [KeycloakRealmRoleBatch](#keycloakrealmrolebatch)

- [KeycloakRealmRole](#keycloakrealmrole)

- [KeycloakRealm](#keycloakrealm)

- [KeycloakRealmUser](#keycloakrealmuser)

- [Keycloak](#keycloak)




## KeycloakAuthFlow
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






KeycloakAuthFlow is the Schema for the keycloak authentication flow API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakAuthFlow</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakauthflowspec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakAuthFlowSpec defines the desired state of KeycloakAuthFlow.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakauthflowstatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakAuthFlowStatus defines the observed state of KeycloakAuthFlow.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakAuthFlow.spec
<sup><sup>[↩ Parent](#keycloakauthflow)</sup></sup>



KeycloakAuthFlowSpec defines the desired state of KeycloakAuthFlow.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          Alias is display name for authentication flow.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>builtIn</b></td>
        <td>boolean</td>
        <td>
          BuiltIn is true if this is built-in auth flow.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>providerId</b></td>
        <td>string</td>
        <td>
          ProviderID for root auth flow and provider for child auth flows.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>topLevel</b></td>
        <td>boolean</td>
        <td>
          TopLevel is true if this is root auth flow.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakauthflowspecauthenticationexecutionsindex">authenticationExecutions</a></b></td>
        <td>[]object</td>
        <td>
          AuthenticationExecutions is list of authentication executions for this auth flow.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>childType</b></td>
        <td>string</td>
        <td>
          ChildType is type for auth flow if it has a parent, available options: basic-flow, form-flow<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description is description for authentication flow.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>parentName</b></td>
        <td>string</td>
        <td>
          ParentName is name of parent auth flow.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakAuthFlow.spec.authenticationExecutions[index]
<sup><sup>[↩ Parent](#keycloakauthflowspec)</sup></sup>



AuthenticationExecution defines keycloak authentication execution.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          Alias is display name for this execution.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>authenticator</b></td>
        <td>string</td>
        <td>
          Authenticator is name of authenticator.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakauthflowspecauthenticationexecutionsindexauthenticatorconfig">authenticatorConfig</a></b></td>
        <td>object</td>
        <td>
          AuthenticatorConfig is configuration for authenticator.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>authenticatorFlow</b></td>
        <td>boolean</td>
        <td>
          AuthenticatorFlow is true if this is auth flow.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>priority</b></td>
        <td>integer</td>
        <td>
          Priority is priority for this execution. Lower values have higher priority.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requirement</b></td>
        <td>string</td>
        <td>
          Requirement is requirement for this execution. Available options: REQUIRED, ALTERNATIVE, DISABLED, CONDITIONAL.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakAuthFlow.spec.authenticationExecutions[index].authenticatorConfig
<sup><sup>[↩ Parent](#keycloakauthflowspecauthenticationexecutionsindex)</sup></sup>



AuthenticatorConfig is configuration for authenticator.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          Alias is display name for authenticator config.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          Config is configuration for authenticator.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakAuthFlow.status
<sup><sup>[↩ Parent](#keycloakauthflow)</sup></sup>



KeycloakAuthFlowStatus defines the observed state of KeycloakAuthFlow.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakClient
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






KeycloakClient is the Schema for the keycloak clients API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakClient</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakClientSpec defines the desired state of KeycloakClient.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientstatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakClientStatus defines the observed state of KeycloakClient.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec
<sup><sup>[↩ Parent](#keycloakclient)</sup></sup>



KeycloakClientSpec defines the desired state of KeycloakClient.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientId</b></td>
        <td>string</td>
        <td>
          ClientId is a unique keycloak client ID referenced in URI and tokens.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>advancedProtocolMappers</b></td>
        <td>boolean</td>
        <td>
          AdvancedProtocolMappers is a flag to enable advanced protocol mappers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string]string</td>
        <td>
          Attributes is a map of client attributes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clientRoles</b></td>
        <td>[]string</td>
        <td>
          ClientRoles is a list of client roles names assigned to client.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>defaultClientScopes</b></td>
        <td>[]string</td>
        <td>
          DefaultClientScopes is a list of default client scopes assigned to client.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>directAccess</b></td>
        <td>boolean</td>
        <td>
          DirectAccess is a flag to set client as direct access.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>frontChannelLogout</b></td>
        <td>boolean</td>
        <td>
          FrontChannelLogout is a flag to enable front channel logout.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          Protocol is a client protocol.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecprotocolmappersindex">protocolMappers</a></b></td>
        <td>[]object</td>
        <td>
          ProtocolMappers is a list of protocol mappers assigned to client.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>public</b></td>
        <td>boolean</td>
        <td>
          Public is a flag to set client as public.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecrealmrolesindex">realmRoles</a></b></td>
        <td>[]object</td>
        <td>
          RealmRoles is a list of realm roles assigned to client.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reconciliationStrategy</b></td>
        <td>enum</td>
        <td>
          ReconciliationStrategy is a strategy to reconcile client.<br/>
          <br/>
            <i>Enum</i>: full, addOnly<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secret</b></td>
        <td>string</td>
        <td>
          Secret is a client secret used for authentication. If not provided, it will be generated.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecserviceaccount">serviceAccount</a></b></td>
        <td>object</td>
        <td>
          ServiceAccount is a service account configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetRealm</b></td>
        <td>string</td>
        <td>
          TargetRealm is a realm name where client will be created.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>webUrl</b></td>
        <td>string</td>
        <td>
          WebUrl is a client web url.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.protocolMappers[index]
<sup><sup>[↩ Parent](#keycloakclientspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          Config is a map of protocol mapper configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is a protocol mapper name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          Protocol is a protocol name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocolMapper</b></td>
        <td>string</td>
        <td>
          ProtocolMapper is a protocol mapper name.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.realmRoles[index]
<sup><sup>[↩ Parent](#keycloakclientspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>composite</b></td>
        <td>string</td>
        <td>
          Composite is a realm composite role name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is a realm role name.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.serviceAccount
<sup><sup>[↩ Parent](#keycloakclientspec)</sup></sup>



ServiceAccount is a service account configuration.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>attributes</b></td>
        <td>map[string]string</td>
        <td>
          Attributes is a map of service account attributes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecserviceaccountclientrolesindex">clientRoles</a></b></td>
        <td>[]object</td>
        <td>
          ClientRoles is a list of client roles assigned to service account.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Enabled is a flag to enable service account.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>realmRoles</b></td>
        <td>[]string</td>
        <td>
          RealmRoles is a list of realm roles assigned to service account.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.serviceAccount.clientRoles[index]
<sup><sup>[↩ Parent](#keycloakclientspecserviceaccount)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientId</b></td>
        <td>string</td>
        <td>
          ClientID is a client ID.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>roles</b></td>
        <td>[]string</td>
        <td>
          Roles is a list of client roles names assigned to service account.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.status
<sup><sup>[↩ Parent](#keycloakclient)</sup></sup>



KeycloakClientStatus defines the observed state of KeycloakClient.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clientSecretName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakClientScope
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






KeycloakClientScope is the Schema for the keycloakclientscopes API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakClientScope</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakclientscopespec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakClientScopeSpec defines the desired state of KeycloakClientScope.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientscopestatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakClientScopeStatus defines the observed state of KeycloakClientScope.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClientScope.spec
<sup><sup>[↩ Parent](#keycloakclientscope)</sup></sup>



KeycloakClientScopeSpec defines the desired state of KeycloakClientScope.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of keycloak client scope.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          Protocol is SSO protocol configuration which is being supplied by this client scope.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string]string</td>
        <td>
          Attributes is a map of client scope attributes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>default</b></td>
        <td>boolean</td>
        <td>
          Default is a flag to set client scope as default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description is a description of client scope.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientscopespecprotocolmappersindex">protocolMappers</a></b></td>
        <td>[]object</td>
        <td>
          ProtocolMappers is a list of protocol mappers assigned to client scope.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClientScope.spec.protocolMappers[index]
<sup><sup>[↩ Parent](#keycloakclientscopespec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          Config is a map of protocol mapper configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is a protocol mapper name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>string</td>
        <td>
          Protocol is a protocol name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocolMapper</b></td>
        <td>string</td>
        <td>
          ProtocolMapper is a protocol mapper name.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClientScope.status
<sup><sup>[↩ Parent](#keycloakclientscope)</sup></sup>



KeycloakClientScopeStatus defines the observed state of KeycloakClientScope.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmComponent
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






KeycloakRealmComponent is the Schema for the keycloak component API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmComponent</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmcomponentspec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakComponentSpec defines the desired state of KeycloakRealmComponent.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmcomponentstatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakComponentStatus defines the observed state of KeycloakRealmComponent.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmComponent.spec
<sup><sup>[↩ Parent](#keycloakrealmcomponent)</sup></sup>



KeycloakComponentSpec defines the desired state of KeycloakRealmComponent.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of keycloak component.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>providerId</b></td>
        <td>string</td>
        <td>
          ProviderID is a provider ID of component.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>providerType</b></td>
        <td>string</td>
        <td>
          ProviderType is a provider type of component.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>config</b></td>
        <td>map[string][]string</td>
        <td>
          Config is a map of component configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmcomponentspecparentref">parentRef</a></b></td>
        <td>object</td>
        <td>
          ParentRef specifies a parent resource. If not specified, then parent is realm specified in realm field.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmComponent.spec.parentRef
<sup><sup>[↩ Parent](#keycloakrealmcomponentspec)</sup></sup>



ParentRef specifies a parent resource. If not specified, then parent is realm specified in realm field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is a name of parent component custom resource. For example, if Kind is KeycloakRealm, then Name is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind is a kind of parent component. By default, it is KeycloakRealm.<br/>
          <br/>
            <i>Enum</i>: KeycloakRealm, KeycloakRealmComponent<br/>
            <i>Default</i>: KeycloakRealm<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmComponent.status
<sup><sup>[↩ Parent](#keycloakrealmcomponent)</sup></sup>



KeycloakComponentStatus defines the observed state of KeycloakRealmComponent.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmGroup
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






KeycloakRealmGroup is the Schema for the keycloak group API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmGroup</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmgroupspec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmGroupSpec defines the desired state of KeycloakRealmGroup.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmgroupstatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmGroupStatus defines the observed state of KeycloakRealmGroup.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmGroup.spec
<sup><sup>[↩ Parent](#keycloakrealmgroup)</sup></sup>



KeycloakRealmGroupSpec defines the desired state of KeycloakRealmGroup.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of keycloak group.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>access</b></td>
        <td>map[string]boolean</td>
        <td>
          Access is a map of group access.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string][]string</td>
        <td>
          Attributes is a map of group attributes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmgroupspecclientrolesindex">clientRoles</a></b></td>
        <td>[]object</td>
        <td>
          ClientRoles is a list of client roles assigned to group.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Path is a group path.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>realmRoles</b></td>
        <td>[]string</td>
        <td>
          RealmRoles is a list of realm roles assigned to group.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subGroups</b></td>
        <td>[]string</td>
        <td>
          SubGroups is a list of subgroups assigned to group.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmGroup.spec.clientRoles[index]
<sup><sup>[↩ Parent](#keycloakrealmgroupspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clientId</b></td>
        <td>string</td>
        <td>
          ClientID is a client ID.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>roles</b></td>
        <td>[]string</td>
        <td>
          Roles is a list of client roles names assigned to service account.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmGroup.status
<sup><sup>[↩ Parent](#keycloakrealmgroup)</sup></sup>



KeycloakRealmGroupStatus defines the observed state of KeycloakRealmGroup.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          ID is a group ID.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmIdentityProvider
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






KeycloakRealmIdentityProvider is the Schema for the keycloak realm identity provider API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmIdentityProvider</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmidentityproviderspec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmIdentityProviderSpec defines the desired state of KeycloakRealmIdentityProvider.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmidentityproviderstatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmIdentityProviderStatus defines the observed state of KeycloakRealmIdentityProvider.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmIdentityProvider.spec
<sup><sup>[↩ Parent](#keycloakrealmidentityprovider)</sup></sup>



KeycloakRealmIdentityProviderSpec defines the desired state of KeycloakRealmIdentityProvider.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          Alias is a alias of identity provider.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          Config is a map of identity provider configuration.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Enabled is a flag to enable/disable identity provider.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>providerId</b></td>
        <td>string</td>
        <td>
          ProviderID is a provider ID of identity provider.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>addReadTokenRoleOnCreate</b></td>
        <td>boolean</td>
        <td>
          AddReadTokenRoleOnCreate is a flag to add read token role on create.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>authenticateByDefault</b></td>
        <td>boolean</td>
        <td>
          AuthenticateByDefault is a flag to authenticate by default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>displayName</b></td>
        <td>string</td>
        <td>
          DisplayName is a display name of identity provider.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>firstBrokerLoginFlowAlias</b></td>
        <td>string</td>
        <td>
          FirstBrokerLoginFlowAlias is a first broker login flow alias.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>linkOnly</b></td>
        <td>boolean</td>
        <td>
          LinkOnly is a flag to link only.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmidentityproviderspecmappersindex">mappers</a></b></td>
        <td>[]object</td>
        <td>
          Mappers is a list of identity provider mappers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storeToken</b></td>
        <td>boolean</td>
        <td>
          StoreToken is a flag to store token.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>trustEmail</b></td>
        <td>boolean</td>
        <td>
          TrustEmail is a flag to trust email.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmIdentityProvider.spec.mappers[index]
<sup><sup>[↩ Parent](#keycloakrealmidentityproviderspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          Config is a map of identity provider mapper configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>identityProviderAlias</b></td>
        <td>string</td>
        <td>
          IdentityProviderAlias is a identity provider alias.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>identityProviderMapper</b></td>
        <td>string</td>
        <td>
          IdentityProviderMapper is a identity provider mapper.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is a name of identity provider mapper.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmIdentityProvider.status
<sup><sup>[↩ Parent](#keycloakrealmidentityprovider)</sup></sup>



KeycloakRealmIdentityProviderStatus defines the observed state of KeycloakRealmIdentityProvider.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmRoleBatch
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






KeycloakRealmRoleBatch is the Schema for the keycloak roles API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmRoleBatch</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolebatchspec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmRoleBatchSpec defines the desired state of KeycloakRealmRoleBatch.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolebatchstatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmRoleBatchStatus defines the observed state of KeycloakRealmRoleBatch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmRoleBatch.spec
<sup><sup>[↩ Parent](#keycloakrealmrolebatch)</sup></sup>



KeycloakRealmRoleBatchSpec defines the desired state of KeycloakRealmRoleBatch.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolebatchspecrolesindex">roles</a></b></td>
        <td>[]object</td>
        <td>
          Roles is a list of roles to be created.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealmRoleBatch.spec.roles[index]
<sup><sup>[↩ Parent](#keycloakrealmrolebatchspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of keycloak role.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string][]string</td>
        <td>
          Attributes is a map of role attributes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>composite</b></td>
        <td>boolean</td>
        <td>
          Composite is a flag if role is composite.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolebatchspecrolesindexcompositesindex">composites</a></b></td>
        <td>[]object</td>
        <td>
          Composites is a list of composites roles assigned to role.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description is a role description.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>isDefault</b></td>
        <td>boolean</td>
        <td>
          IsDefault is a flag if role is default.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmRoleBatch.spec.roles[index].composites[index]
<sup><sup>[↩ Parent](#keycloakrealmrolebatchspecrolesindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is a name of composite role.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealmRoleBatch.status
<sup><sup>[↩ Parent](#keycloakrealmrolebatch)</sup></sup>



KeycloakRealmRoleBatchStatus defines the observed state of KeycloakRealmRoleBatch.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmRole
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






KeycloakRealmRole is the Schema for the keycloak group API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmRole</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolespec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmRoleSpec defines the desired state of KeycloakRealmRole.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolestatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmRoleStatus defines the observed state of KeycloakRealmRole.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmRole.spec
<sup><sup>[↩ Parent](#keycloakrealmrole)</sup></sup>



KeycloakRealmRoleSpec defines the desired state of KeycloakRealmRole.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of keycloak role.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string][]string</td>
        <td>
          Attributes is a map of role attributes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>composite</b></td>
        <td>boolean</td>
        <td>
          Composite is a flag if role is composite.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolespeccompositesindex">composites</a></b></td>
        <td>[]object</td>
        <td>
          Composites is a list of composites roles assigned to role.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description is a role description.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>isDefault</b></td>
        <td>boolean</td>
        <td>
          IsDefault is a flag if role is default.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmRole.spec.composites[index]
<sup><sup>[↩ Parent](#keycloakrealmrolespec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is a name of composite role.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealmRole.status
<sup><sup>[↩ Parent](#keycloakrealmrole)</sup></sup>



KeycloakRealmRoleStatus defines the observed state of KeycloakRealmRole.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          ID is a role ID.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealm
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






KeycloakRealm is the Schema for the keycloak realms API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealm</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmSpec defines the desired state of KeycloakRealm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmstatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmStatus defines the observed state of KeycloakRealm.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec
<sup><sup>[↩ Parent](#keycloakrealm)</sup></sup>



KeycloakRealmSpec defines the desired state of KeycloakRealm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>realmName</b></td>
        <td>string</td>
        <td>
          RealmName specifies the name of the realm.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>browserFlow</b></td>
        <td>string</td>
        <td>
          BrowserFlow specifies the authentication flow to use for the realm's browser clients.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>browserSecurityHeaders</b></td>
        <td>map[string]string</td>
        <td>
          BrowserSecurityHeaders is a map of security headers to apply to HTTP responses from the realm's browser clients.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>disableCentralIDPMappers</b></td>
        <td>boolean</td>
        <td>
          DisableCentralIDPMappers indicates whether to disable the default identity provider (IDP) mappers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>frontendUrl</b></td>
        <td>string</td>
        <td>
          FrontendURL Set the frontend URL for the realm. Use in combination with the default hostname provider to override the base URL for frontend requests for a specific realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          ID is the ID of the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>keycloakOwner</b></td>
        <td>string</td>
        <td>
          KeycloakOwner specifies the name of the Keycloak instance that owns the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecpasswordpolicyindex">passwordPolicy</a></b></td>
        <td>[]object</td>
        <td>
          PasswordPolicies is a list of password policies to apply to the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecrealmeventconfig">realmEventConfig</a></b></td>
        <td>object</td>
        <td>
          RealmEventConfig is the configuration for events in the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ssoAutoRedirectEnabled</b></td>
        <td>boolean</td>
        <td>
          SsoAutoRedirectEnabled indicates whether to enable automatic redirection to the SSO realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ssoRealmEnabled</b></td>
        <td>boolean</td>
        <td>
          SsoRealmEnabled indicates whether to enable the SSO realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecssorealmmappersindex">ssoRealmMappers</a></b></td>
        <td>[]object</td>
        <td>
          SSORealmMappers is a list of SSO realm mappers to create in the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ssoRealmName</b></td>
        <td>string</td>
        <td>
          SsoRealmName specifies the name of the SSO realm used by the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecthemes">themes</a></b></td>
        <td>object</td>
        <td>
          Themes is a map of themes to apply to the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecusersindex">users</a></b></td>
        <td>[]object</td>
        <td>
          Users is a list of users to create in the realm.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.passwordPolicy[index]
<sup><sup>[↩ Parent](#keycloakrealmspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type of password policy.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value of password policy.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.realmEventConfig
<sup><sup>[↩ Parent](#keycloakrealmspec)</sup></sup>



RealmEventConfig is the configuration for events in the realm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>adminEventsDetailsEnabled</b></td>
        <td>boolean</td>
        <td>
          AdminEventsDetailsEnabled indicates whether to enable detailed admin events.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>adminEventsEnabled</b></td>
        <td>boolean</td>
        <td>
          AdminEventsEnabled indicates whether to enable admin events.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabledEventTypes</b></td>
        <td>[]string</td>
        <td>
          EnabledEventTypes is a list of event types to enable.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>eventsEnabled</b></td>
        <td>boolean</td>
        <td>
          EventsEnabled indicates whether to enable events.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>eventsExpiration</b></td>
        <td>integer</td>
        <td>
          EventsExpiration is the number of seconds after which events expire.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>eventsListeners</b></td>
        <td>[]string</td>
        <td>
          EventsListeners is a list of event listeners to enable.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.ssoRealmMappers[index]
<sup><sup>[↩ Parent](#keycloakrealmspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>config</b></td>
        <td>map[string]string</td>
        <td>
          Config is a map of configuration options for the SSO realm mapper.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>identityProviderMapper</b></td>
        <td>string</td>
        <td>
          IdentityProviderMapper specifies the identity provider mapper to use.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the SSO realm mapper.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.themes
<sup><sup>[↩ Parent](#keycloakrealmspec)</sup></sup>



Themes is a map of themes to apply to the realm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accountTheme</b></td>
        <td>string</td>
        <td>
          AccountTheme specifies the account theme to use for the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>adminConsoleTheme</b></td>
        <td>string</td>
        <td>
          AdminConsoleTheme specifies the admin console theme to use for the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>emailTheme</b></td>
        <td>string</td>
        <td>
          EmailTheme specifies the email theme to use for the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>internationalizationEnabled</b></td>
        <td>boolean</td>
        <td>
          InternationalizationEnabled indicates whether to enable internationalization.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>loginTheme</b></td>
        <td>string</td>
        <td>
          LoginTheme specifies the login theme to use for the realm.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.users[index]
<sup><sup>[↩ Parent](#keycloakrealmspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>username</b></td>
        <td>string</td>
        <td>
          Username of keycloak user.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realmRoles</b></td>
        <td>[]string</td>
        <td>
          RealmRoles is a list of roles attached to keycloak user.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.status
<sup><sup>[↩ Parent](#keycloakrealm)</sup></sup>



KeycloakRealmStatus defines the observed state of KeycloakRealm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>available</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## KeycloakRealmUser
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






KeycloakRealmUser is the Schema for the keycloak user API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>KeycloakRealmUser</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmuserspec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmUserSpec defines the desired state of KeycloakRealmUser.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmuserstatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakRealmUserStatus defines the observed state of KeycloakRealmUser.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmUser.spec
<sup><sup>[↩ Parent](#keycloakrealmuser)</sup></sup>



KeycloakRealmUserSpec defines the desired state of KeycloakRealmUser.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>username</b></td>
        <td>string</td>
        <td>
          Username is a username in keycloak.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string]string</td>
        <td>
          Attributes is a map of user attributes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>email</b></td>
        <td>string</td>
        <td>
          Email is a user email.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>emailVerified</b></td>
        <td>boolean</td>
        <td>
          EmailVerified is a user email verified flag.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Enabled is a user enabled flag.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>firstName</b></td>
        <td>string</td>
        <td>
          FirstName is a user first name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groups</b></td>
        <td>[]string</td>
        <td>
          Groups is a list of groups assigned to user.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>keepResource</b></td>
        <td>boolean</td>
        <td>
          KeepResource is a flag if resource should be kept after deletion. If set to true, user will not be deleted from keycloak.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>lastName</b></td>
        <td>string</td>
        <td>
          LastName is a user last name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>password</b></td>
        <td>string</td>
        <td>
          Password is a user password. Allows to keep user password within Custom Resource. For security concerns, it is recommended to use PasswordSecret instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmuserspecpasswordsecret">passwordSecret</a></b></td>
        <td>object</td>
        <td>
          PasswordSecret defines Kubernetes secret Name and Key, which holds User secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reconciliationStrategy</b></td>
        <td>string</td>
        <td>
          ReconciliationStrategy is a strategy for reconciliation. Possible values: full, create-only. Default value: full. If set to create-only, user will be created only if it does not exist. If user exists, it will not be updated. If set to full, user will be created if it does not exist, or updated if it exists.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requiredUserActions</b></td>
        <td>[]string</td>
        <td>
          RequiredUserActions is required action when user log in, example: CONFIGURE_TOTP, UPDATE_PASSWORD, UPDATE_PROFILE, VERIFY_EMAIL.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>roles</b></td>
        <td>[]string</td>
        <td>
          Roles is a list of roles assigned to user.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmUser.spec.passwordSecret
<sup><sup>[↩ Parent](#keycloakrealmuserspec)</sup></sup>



PasswordSecret defines Kubernetes secret Name and Key, which holds User secret.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the key in the secret.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of the secret.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealmUser.status
<sup><sup>[↩ Parent](#keycloakrealmuser)</sup></sup>



KeycloakRealmUserStatus defines the observed state of KeycloakRealmUser.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Keycloak
<sup><sup>[↩ Parent](#v1edpepamcomv1 )</sup></sup>






Keycloak is the Schema for the keycloaks API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v1.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Keycloak</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakspec">spec</a></b></td>
        <td>object</td>
        <td>
          KeycloakSpec defines the desired state of Keycloak.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakstatus">status</a></b></td>
        <td>object</td>
        <td>
          KeycloakStatus defines the observed state of Keycloak.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Keycloak.spec
<sup><sup>[↩ Parent](#keycloak)</sup></sup>



KeycloakSpec defines the desired state of Keycloak.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>secret</b></td>
        <td>string</td>
        <td>
          Secret is a secret name which contains admin credentials.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          URL of keycloak service.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>adminType</b></td>
        <td>enum</td>
        <td>
          AdminType can be user or serviceAccount, if serviceAccount was specified, then client_credentials grant type should be used for getting admin realm token.<br/>
          <br/>
            <i>Enum</i>: serviceAccount, user<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Keycloak.status
<sup><sup>[↩ Parent](#keycloak)</sup></sup>



KeycloakStatus defines the observed state of Keycloak.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>connected</b></td>
        <td>boolean</td>
        <td>
          Connected shows if keycloak service is up and running.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>