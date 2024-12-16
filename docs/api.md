# API Reference

Packages:

- [v1.edp.epam.com/v1alpha1](#v1edpepamcomv1alpha1)
- [v1.edp.epam.com/v1](#v1edpepamcomv1)

# v1.edp.epam.com/v1alpha1

Resource Types:

- [ClusterKeycloakRealm](#clusterkeycloakrealm)

- [ClusterKeycloak](#clusterkeycloak)




## ClusterKeycloakRealm
<sup><sup>[↩ Parent](#v1edpepamcomv1alpha1 )</sup></sup>






ClusterKeycloakRealm is the Schema for the clusterkeycloakrealms API.

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
      <td>ClusterKeycloakRealm</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspec">spec</a></b></td>
        <td>object</td>
        <td>
          ClusterKeycloakRealmSpec defines the desired state of ClusterKeycloakRealm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmstatus">status</a></b></td>
        <td>object</td>
        <td>
          ClusterKeycloakRealmStatus defines the observed state of ClusterKeycloakRealm.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec
<sup><sup>[↩ Parent](#clusterkeycloakrealm)</sup></sup>



ClusterKeycloakRealmSpec defines the desired state of ClusterKeycloakRealm.

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
        <td><b>clusterKeycloakRef</b></td>
        <td>string</td>
        <td>
          ClusterKeycloakRef is a name of the ClusterKeycloak instance that owns the realm.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realmName</b></td>
        <td>string</td>
        <td>
          RealmName specifies the name of the realm.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecauthenticationflows">authenticationFlows</a></b></td>
        <td>object</td>
        <td>
          AuthenticationFlow is the configuration for authentication flows in the realm.<br/>
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
        <td><b>displayHtmlName</b></td>
        <td>string</td>
        <td>
          DisplayHTMLName name to render in the UI.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>displayName</b></td>
        <td>string</td>
        <td>
          DisplayName is the display name of the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>frontendUrl</b></td>
        <td>string</td>
        <td>
          FrontendURL Set the frontend URL for the realm.
Use in combination with the default hostname provider to override the base URL for frontend requests for a specific realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspeclocalization">localization</a></b></td>
        <td>object</td>
        <td>
          Localization is the configuration for localization in the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecpasswordpolicyindex">passwordPolicy</a></b></td>
        <td>[]object</td>
        <td>
          PasswordPolicies is a list of password policies to apply to the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecrealmeventconfig">realmEventConfig</a></b></td>
        <td>object</td>
        <td>
          RealmEventConfig is the configuration for events in the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecsmtp">smtp</a></b></td>
        <td>object</td>
        <td>
          Smtp is the configuration for email in the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecthemes">themes</a></b></td>
        <td>object</td>
        <td>
          Themes is a map of themes to apply to the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspectokensettings">tokenSettings</a></b></td>
        <td>object</td>
        <td>
          TokenSettings is the configuration for tokens in the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecuserprofileconfig">userProfileConfig</a></b></td>
        <td>object</td>
        <td>
          UserProfileConfig is the configuration for user profiles in the realm.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.authenticationFlows
<sup><sup>[↩ Parent](#clusterkeycloakrealmspec)</sup></sup>



AuthenticationFlow is the configuration for authentication flows in the realm.

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
        <td><b>browserFlow</b></td>
        <td>string</td>
        <td>
          BrowserFlow specifies the authentication flow to use for the realm's browser clients.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.localization
<sup><sup>[↩ Parent](#clusterkeycloakrealmspec)</sup></sup>



Localization is the configuration for localization in the realm.

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
        <td><b>internationalizationEnabled</b></td>
        <td>boolean</td>
        <td>
          InternationalizationEnabled indicates whether to enable internationalization.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.passwordPolicy[index]
<sup><sup>[↩ Parent](#clusterkeycloakrealmspec)</sup></sup>





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


### ClusterKeycloakRealm.spec.realmEventConfig
<sup><sup>[↩ Parent](#clusterkeycloakrealmspec)</sup></sup>



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


### ClusterKeycloakRealm.spec.smtp
<sup><sup>[↩ Parent](#clusterkeycloakrealmspec)</sup></sup>



Smtp is the configuration for email in the realm.

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
        <td><b><a href="#clusterkeycloakrealmspecsmtpconnection">connection</a></b></td>
        <td>object</td>
        <td>
          Connection specifies the email connection configuration.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecsmtptemplate">template</a></b></td>
        <td>object</td>
        <td>
          Template specifies the email template configuration.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.smtp.connection
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecsmtp)</sup></sup>



Connection specifies the email connection configuration.

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
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host specifies the email server host.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecsmtpconnectionauthentication">authentication</a></b></td>
        <td>object</td>
        <td>
          Authentication specifies the email authentication configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableSSL</b></td>
        <td>boolean</td>
        <td>
          EnableSSL specifies if SSL is enabled.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableStartTLS</b></td>
        <td>boolean</td>
        <td>
          EnableStartTLS specifies if StartTLS is enabled.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port specifies the email server port.<br/>
          <br/>
            <i>Default</i>: 25<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.smtp.connection.authentication
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecsmtpconnection)</sup></sup>



Authentication specifies the email authentication configuration.

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
        <td><b><a href="#clusterkeycloakrealmspecsmtpconnectionauthenticationpassword">password</a></b></td>
        <td>object</td>
        <td>
          Password specifies login password.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecsmtpconnectionauthenticationusername">username</a></b></td>
        <td>object</td>
        <td>
          Username specifies login username.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.smtp.connection.authentication.password
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecsmtpconnectionauthentication)</sup></sup>



Password specifies login password.

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
        <td><b><a href="#clusterkeycloakrealmspecsmtpconnectionauthenticationpasswordconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecsmtpconnectionauthenticationpasswordsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.smtp.connection.authentication.password.configMapKeyRef
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecsmtpconnectionauthenticationpassword)</sup></sup>



Selects a key of a ConfigMap.

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
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.smtp.connection.authentication.password.secretKeyRef
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecsmtpconnectionauthenticationpassword)</sup></sup>



Selects a key of a secret.

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
          The key of the secret to select from.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.smtp.connection.authentication.username
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecsmtpconnectionauthentication)</sup></sup>



Username specifies login username.

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
        <td><b><a href="#clusterkeycloakrealmspecsmtpconnectionauthenticationusernameconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecsmtpconnectionauthenticationusernamesecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Directly specifies a value.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.smtp.connection.authentication.username.configMapKeyRef
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecsmtpconnectionauthenticationusername)</sup></sup>



Selects a key of a ConfigMap.

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
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.smtp.connection.authentication.username.secretKeyRef
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecsmtpconnectionauthenticationusername)</sup></sup>



Selects a key of a secret.

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
          The key of the secret to select from.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.smtp.template
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecsmtp)</sup></sup>



Template specifies the email template configuration.

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
        <td><b>from</b></td>
        <td>string</td>
        <td>
          From specifies the sender email address.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>envelopeFrom</b></td>
        <td>string</td>
        <td>
          EnvelopeFrom is an email address used for bounces .<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>fromDisplayName</b></td>
        <td>string</td>
        <td>
          FromDisplayName specifies the sender display for sender email address.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replyTo</b></td>
        <td>string</td>
        <td>
          ReplyTo specifies the reply-to email address.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replyToDisplayName</b></td>
        <td>string</td>
        <td>
          ReplyToDisplayName specifies display name for reply-to email address.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.themes
<sup><sup>[↩ Parent](#clusterkeycloakrealmspec)</sup></sup>



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
        <td><b>loginTheme</b></td>
        <td>string</td>
        <td>
          LoginTheme specifies the login theme to use for the realm.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.tokenSettings
<sup><sup>[↩ Parent](#clusterkeycloakrealmspec)</sup></sup>



TokenSettings is the configuration for tokens in the realm.

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
        <td><b>accessCodeLifespan</b></td>
        <td>integer</td>
        <td>
          AccessCodeLifespan specifies max time(in seconds)a client has to finish the access token protocol.
This should normally be 1 minute.<br/>
          <br/>
            <i>Default</i>: 60<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>accessToken</b></td>
        <td>integer</td>
        <td>
          AccessTokenLifespanForImplicitFlow specifies max time(in seconds) before an access token is expired for implicit flow.<br/>
          <br/>
            <i>Default</i>: 900<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>accessTokenLifespan</b></td>
        <td>integer</td>
        <td>
          AccessTokenLifespan specifies max time(in seconds) before an access token is expired.
This value is recommended to be short relative to the SSO timeout.<br/>
          <br/>
            <i>Default</i>: 300<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>actionTokenGeneratedByAdminLifespan</b></td>
        <td>integer</td>
        <td>
          ActionTokenGeneratedByAdminLifespan specifies max time(in seconds) before an action permit sent to a user by administrator is expired.
This value is recommended to be long to allow administrators to send e-mails for users that are currently offline.
The default timeout can be overridden immediately before issuing the token.<br/>
          <br/>
            <i>Default</i>: 43200<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>actionTokenGeneratedByUserLifespan</b></td>
        <td>integer</td>
        <td>
          AccessCodeLifespanUserAction specifies max time(in seconds) before an action permit sent by a user (such as a forgot password e-mail) is expired.
This value is recommended to be short because it's expected that the user would react to self-created action quickly.<br/>
          <br/>
            <i>Default</i>: 300<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>defaultSignatureAlgorithm</b></td>
        <td>enum</td>
        <td>
          DefaultSignatureAlgorithm specifies the default algorithm used to sign tokens for the realm<br/>
          <br/>
            <i>Enum</i>: ES256, ES384, ES512, EdDSA, HS256, HS384, HS512, PS256, PS384, PS512, RS256, RS384, RS512<br/>
            <i>Default</i>: RS256<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>refreshTokenMaxReuse</b></td>
        <td>integer</td>
        <td>
          RefreshTokenMaxReuse specifies maximum number of times a refresh token can be reused.
When a different token is used, revocation is immediate.<br/>
          <br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>revokeRefreshToken</b></td>
        <td>boolean</td>
        <td>
          RevokeRefreshToken if enabled a refresh token can only be used up to 'refreshTokenMaxReuse' and
is revoked when a different token is used.
Otherwise, refresh tokens are not revoked when used and can be used multiple times.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.userProfileConfig
<sup><sup>[↩ Parent](#clusterkeycloakrealmspec)</sup></sup>



UserProfileConfig is the configuration for user profiles in the realm.

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
        <td><b><a href="#clusterkeycloakrealmspecuserprofileconfigattributesindex">attributes</a></b></td>
        <td>[]object</td>
        <td>
          Attributes specifies the list of user profile attributes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecuserprofileconfiggroupsindex">groups</a></b></td>
        <td>[]object</td>
        <td>
          Groups specifies the list of user profile groups.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>unmanagedAttributePolicy</b></td>
        <td>string</td>
        <td>
          UnmanagedAttributePolicy are user attributes not explicitly defined in the user profile configuration.
Empty value means that unmanaged attributes are disabled.
Possible values:
ENABLED - unmanaged attributes are allowed.
ADMIN_VIEW - unmanaged attributes are read-only and only available through the administration console and API.
ADMIN_EDIT - unmanaged attributes can be managed only through the administration console and API.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.userProfileConfig.attributes[index]
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecuserprofileconfig)</sup></sup>





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
          Name of the user attribute, used to uniquely identify an attribute.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td>
          Annotations specifies the annotations for the attribute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>displayName</b></td>
        <td>string</td>
        <td>
          Display name for the attribute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>group</b></td>
        <td>string</td>
        <td>
          Group to which the attribute belongs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>multivalued</b></td>
        <td>boolean</td>
        <td>
          Multivalued specifies if this attribute supports multiple values.
This setting is an indicator and does not enable any validation<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecuserprofileconfigattributesindexpermissions">permissions</a></b></td>
        <td>object</td>
        <td>
          Permissions specifies the permissions for the attribute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecuserprofileconfigattributesindexrequired">required</a></b></td>
        <td>object</td>
        <td>
          Required indicates that the attribute must be set by users and administrators.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecuserprofileconfigattributesindexselector">selector</a></b></td>
        <td>object</td>
        <td>
          Selector specifies the scopes for which the attribute is available.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakrealmspecuserprofileconfigattributesindexvalidationskeykey">validations</a></b></td>
        <td>map[string]map[string]object</td>
        <td>
          Validations specifies the validations for the attribute.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.userProfileConfig.attributes[index].permissions
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecuserprofileconfigattributesindex)</sup></sup>



Permissions specifies the permissions for the attribute.

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
        <td><b>edit</b></td>
        <td>[]string</td>
        <td>
          Edit specifies who can edit the attribute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>view</b></td>
        <td>[]string</td>
        <td>
          View specifies who can view the attribute.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.userProfileConfig.attributes[index].required
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecuserprofileconfigattributesindex)</sup></sup>



Required indicates that the attribute must be set by users and administrators.

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
        <td><b>roles</b></td>
        <td>[]string</td>
        <td>
          Roles specifies the roles for whom the attribute is required.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scopes</b></td>
        <td>[]string</td>
        <td>
          Scopes specifies the scopes when the attribute is required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.userProfileConfig.attributes[index].selector
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecuserprofileconfigattributesindex)</sup></sup>



Selector specifies the scopes for which the attribute is available.

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
        <td><b>scopes</b></td>
        <td>[]string</td>
        <td>
          Scopes specifies the scopes for which the attribute is available.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.userProfileConfig.attributes[index].validations[key][key]
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecuserprofileconfigattributesindex)</sup></sup>





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
        <td><b>intVal</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mapVal</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sliceVal</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>stringVal</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.spec.userProfileConfig.groups[index]
<sup><sup>[↩ Parent](#clusterkeycloakrealmspecuserprofileconfig)</sup></sup>





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
          Name is unique name of the group.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td>
          Annotations specifies the annotations for the group.
nullable<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>displayDescription</b></td>
        <td>string</td>
        <td>
          DisplayDescription specifies a user-friendly name for the group that should be used when rendering a group of attributes in user-facing forms.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>displayHeader</b></td>
        <td>string</td>
        <td>
          DisplayHeader specifies a text that should be used as a header when rendering user-facing forms.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloakRealm.status
<sup><sup>[↩ Parent](#clusterkeycloakrealm)</sup></sup>



ClusterKeycloakRealmStatus defines the observed state of ClusterKeycloakRealm.

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
          <br/>
            <i>Default</i>: map[connected:false]<br/>
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
          AdminType can be user or serviceAccount, if serviceAccount was specified,
then client_credentials grant type should be used for getting admin realm token.<br/>
          <br/>
            <i>Enum</i>: serviceAccount, user<br/>
            <i>Default</i>: user<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakspeccacert">caCert</a></b></td>
        <td>object</td>
        <td>
          CACert defines the root certificate authority
that api clients use when verifying server certificates.
Resources should be in the namespace defined in operator OPERATOR_NAMESPACE env.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          InsecureSkipVerify controls whether api client verifies the server's
certificate chain and host name. If InsecureSkipVerify is true, api client
accepts any certificate presented by the server and any host name in that
certificate.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloak.spec.caCert
<sup><sup>[↩ Parent](#clusterkeycloakspec)</sup></sup>



CACert defines the root certificate authority
that api clients use when verifying server certificates.
Resources should be in the namespace defined in operator OPERATOR_NAMESPACE env.

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
        <td><b><a href="#clusterkeycloakspeccacertconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clusterkeycloakspeccacertsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloak.spec.caCert.configMapKeyRef
<sup><sup>[↩ Parent](#clusterkeycloakspeccacert)</sup></sup>



Selects a key of a ConfigMap.

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
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterKeycloak.spec.caCert.secretKeyRef
<sup><sup>[↩ Parent](#clusterkeycloakspeccacert)</sup></sup>



Selects a key of a secret.

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
          The key of the secret to select from.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
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
        <td><b>childRequirement</b></td>
        <td>string</td>
        <td>
          ChildRequirement is requirement for child execution. Available options: REQUIRED, ALTERNATIVE, DISABLED, CONDITIONAL.<br/>
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
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Deprecated: use RealmRef instead.
Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakauthflowspecrealmref">realmRef</a></b></td>
        <td>object</td>
        <td>
          RealmRef is reference to Realm custom resource.<br/>
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


### KeycloakAuthFlow.spec.realmRef
<sup><sup>[↩ Parent](#keycloakauthflowspec)</sup></sup>



RealmRef is reference to Realm custom resource.

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
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind specifies the kind of the Keycloak resource.<br/>
          <br/>
            <i>Enum</i>: KeycloakRealm, ClusterKeycloakRealm<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Keycloak resource.<br/>
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
        <td><b>adminUrl</b></td>
        <td>string</td>
        <td>
          AdminUrl is client admin url.
If empty - WebUrl will be used.<br/>
        </td>
        <td>false</td>
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
          <br/>
            <i>Default</i>: map[post.logout.redirect.uris:+]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecauthorization">authorization</a></b></td>
        <td>object</td>
        <td>
          Authorization is a client authorization configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>authorizationServicesEnabled</b></td>
        <td>boolean</td>
        <td>
          ServiceAccountsEnabled enable/disable fine-grained authorization support for a client.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>bearerOnly</b></td>
        <td>boolean</td>
        <td>
          BearerOnly is a flag to enable bearer-only.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clientAuthenticatorType</b></td>
        <td>string</td>
        <td>
          ClientAuthenticatorType is a client authenticator type.<br/>
          <br/>
            <i>Default</i>: client-secret<br/>
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
        <td><b>consentRequired</b></td>
        <td>boolean</td>
        <td>
          ConsentRequired is a flag to enable consent.<br/>
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
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description is a client description.<br/>
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
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Enabled is a flag to enable client.<br/>
          <br/>
            <i>Default</i>: true<br/>
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
        <td><b>fullScopeAllowed</b></td>
        <td>boolean</td>
        <td>
          FullScopeAllowed is a flag to enable full scope.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>homeUrl</b></td>
        <td>string</td>
        <td>
          HomeUrl is a client home url.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>implicitFlowEnabled</b></td>
        <td>boolean</td>
        <td>
          ImplicitFlowEnabled is a flag to enable support for OpenID Connect redirect based authentication without authorization code.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is a client name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optionalClientScopes</b></td>
        <td>[]string</td>
        <td>
          OptionalClientScopes is a list of optional client scopes assigned to client.<br/>
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
        <td><b><a href="#keycloakclientspecrealmref">realmRef</a></b></td>
        <td>object</td>
        <td>
          RealmRef is reference to Realm custom resource.<br/>
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
        <td><b>redirectUris</b></td>
        <td>[]string</td>
        <td>
          RedirectUris is a list of valid URI pattern a browser can redirect to after a successful login.
Simple wildcards are allowed such as 'https://example.com/*'.
Relative path can be specified too, such as /my/relative/path/*. Relative paths are relative to the client root URL.
If not specified, spec.webUrl + "/*" will be used.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secret</b></td>
        <td>string</td>
        <td>
          Secret is kubernetes secret name where the client's secret will be stored.
Secret should have the following format: $secretName:secretKey.
If not specified, a client secret will be generated and stored in a secret with the name keycloak-client-{metadata.name}-secret.
If keycloak client is public, secret property will be ignored.<br/>
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
        <td><b>standardFlowEnabled</b></td>
        <td>boolean</td>
        <td>
          StandardFlowEnabled is a flag to enable standard flow.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>surrogateAuthRequired</b></td>
        <td>boolean</td>
        <td>
          SurrogateAuthRequired is a flag to enable surrogate auth.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetRealm</b></td>
        <td>string</td>
        <td>
          Deprecated: use RealmRef instead.
TargetRealm is a realm name where client will be created.
It has higher priority than RealmRef for backward compatibility.
If both TargetRealm and RealmRef are specified, TargetRealm will be used for client creation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>webOrigins</b></td>
        <td>[]string</td>
        <td>
          WebOrigins is a list of allowed CORS origins.
To permit all origins of Valid Redirect URIs, add '+'. This does not include the '*' wildcard though.
To permit all origins, explicitly add '*'.
If not specified, the value from `WebUrl` is used<br/>
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


### KeycloakClient.spec.authorization
<sup><sup>[↩ Parent](#keycloakclientspec)</sup></sup>



Authorization is a client authorization configuration.

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
        <td><b><a href="#keycloakclientspecauthorizationpermissionsindex">permissions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecauthorizationpoliciesindex">policies</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecauthorizationresourcesindex">resources</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scopes</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.permissions[index]
<sup><sup>[↩ Parent](#keycloakclientspecauthorization)</sup></sup>





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
          Name is a permission name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type is a permission type.<br/>
          <br/>
            <i>Enum</i>: resource, scope<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>decisionStrategy</b></td>
        <td>enum</td>
        <td>
          DecisionStrategy is a permission decision strategy.<br/>
          <br/>
            <i>Enum</i>: UNANIMOUS, AFFIRMATIVE, CONSENSUS<br/>
            <i>Default</i>: UNANIMOUS<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description is a permission description.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>logic</b></td>
        <td>enum</td>
        <td>
          Logic is a permission logic.<br/>
          <br/>
            <i>Enum</i>: POSITIVE, NEGATIVE<br/>
            <i>Default</i>: POSITIVE<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>policies</b></td>
        <td>[]string</td>
        <td>
          Policies is a list of policies names.
Specifies all the policies that must be applied to the scopes defined by this policy or permission.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>resources</b></td>
        <td>[]string</td>
        <td>
          Resources is a list of resources names.
Specifies that this permission must be applied to all resource instances of a given type.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scopes</b></td>
        <td>[]string</td>
        <td>
          Scopes is a list of authorization scopes names.
Specifies that this permission must be applied to one or more scopes.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.policies[index]
<sup><sup>[↩ Parent](#keycloakclientspecauthorization)</sup></sup>



Policy represents a client authorization policy.

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
          Name is a policy name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type is a policy type.<br/>
          <br/>
            <i>Enum</i>: aggregate, client, group, role, time, user<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecauthorizationpoliciesindexaggregatedpolicy">aggregatedPolicy</a></b></td>
        <td>object</td>
        <td>
          AggregatedPolicy is an aggregated policy settings.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecauthorizationpoliciesindexclientpolicy">clientPolicy</a></b></td>
        <td>object</td>
        <td>
          ClientPolicy is a client policy settings.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>decisionStrategy</b></td>
        <td>enum</td>
        <td>
          DecisionStrategy is a policy decision strategy.<br/>
          <br/>
            <i>Enum</i>: UNANIMOUS, AFFIRMATIVE, CONSENSUS<br/>
            <i>Default</i>: UNANIMOUS<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description is a policy description.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecauthorizationpoliciesindexgrouppolicy">groupPolicy</a></b></td>
        <td>object</td>
        <td>
          GroupPolicy is a group policy settings.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>logic</b></td>
        <td>enum</td>
        <td>
          Logic is a policy logic.<br/>
          <br/>
            <i>Enum</i>: POSITIVE, NEGATIVE<br/>
            <i>Default</i>: POSITIVE<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecauthorizationpoliciesindexrolepolicy">rolePolicy</a></b></td>
        <td>object</td>
        <td>
          RolePolicy is a role policy settings.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecauthorizationpoliciesindextimepolicy">timePolicy</a></b></td>
        <td>object</td>
        <td>
          ScopePolicy is a scope policy settings.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientspecauthorizationpoliciesindexuserpolicy">userPolicy</a></b></td>
        <td>object</td>
        <td>
          UserPolicy is a user policy settings.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.policies[index].aggregatedPolicy
<sup><sup>[↩ Parent](#keycloakclientspecauthorizationpoliciesindex)</sup></sup>



AggregatedPolicy is an aggregated policy settings.

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
        <td><b>policies</b></td>
        <td>[]string</td>
        <td>
          Policies is a list of aggregated policies names.
Specifies all the policies that must be applied to the scopes defined by this policy or permission.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.policies[index].clientPolicy
<sup><sup>[↩ Parent](#keycloakclientspecauthorizationpoliciesindex)</sup></sup>



ClientPolicy is a client policy settings.

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
        <td><b>clients</b></td>
        <td>[]string</td>
        <td>
          Clients is a list of client names. Specifies which client(s) are allowed by this policy.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.policies[index].groupPolicy
<sup><sup>[↩ Parent](#keycloakclientspecauthorizationpoliciesindex)</sup></sup>



GroupPolicy is a group policy settings.

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
        <td><b><a href="#keycloakclientspecauthorizationpoliciesindexgrouppolicygroupsindex">groups</a></b></td>
        <td>[]object</td>
        <td>
          Groups is a list of group names. Specifies which group(s) are allowed by this policy.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groupsClaim</b></td>
        <td>string</td>
        <td>
          GroupsClaim is a group claim.
If defined, the policy will fetch user's groups from the given claim
within an access token or ID token representing the identity asking permissions.
If not defined, user's groups are obtained from your realm configuration.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.policies[index].groupPolicy.groups[index]
<sup><sup>[↩ Parent](#keycloakclientspecauthorizationpoliciesindexgrouppolicy)</sup></sup>



GroupDefinition represents a group in a GroupPolicyData.

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
          Name is a group name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>extendChildren</b></td>
        <td>boolean</td>
        <td>
          ExtendChildren is a flag that specifies whether to extend children.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.policies[index].rolePolicy
<sup><sup>[↩ Parent](#keycloakclientspecauthorizationpoliciesindex)</sup></sup>



RolePolicy is a role policy settings.

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
        <td><b><a href="#keycloakclientspecauthorizationpoliciesindexrolepolicyrolesindex">roles</a></b></td>
        <td>[]object</td>
        <td>
          Roles is a list of role.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.policies[index].rolePolicy.roles[index]
<sup><sup>[↩ Parent](#keycloakclientspecauthorizationpoliciesindexrolepolicy)</sup></sup>



RoleDefinition represents a role in a RolePolicyData.

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
          Name is a role name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>required</b></td>
        <td>boolean</td>
        <td>
          Required is a flag that specifies whether the role is required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.policies[index].timePolicy
<sup><sup>[↩ Parent](#keycloakclientspecauthorizationpoliciesindex)</sup></sup>



ScopePolicy is a scope policy settings.

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
        <td><b>notBefore</b></td>
        <td>string</td>
        <td>
          NotBefore defines the time before which the policy MUST NOT be granted.
Only granted if current date/time is after or equal to this value.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notOnOrAfter</b></td>
        <td>string</td>
        <td>
          NotOnOrAfter defines the time after which the policy MUST NOT be granted.
Only granted if current date/time is before or equal to this value.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>dayMonth</b></td>
        <td>string</td>
        <td>
          Day defines the month which the policy MUST be granted.
You can also provide a range by filling the dayMonthEnd field.
In this case, permission is granted only if current month is between or equal to the two values you provided.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>dayMonthEnd</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hour</b></td>
        <td>string</td>
        <td>
          Hour defines the hour when the policy MUST be granted.
You can also provide a range by filling the hourEnd.
In this case, permission is granted only if current hour is between or equal to the two values you provided.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hourEnd</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minute</b></td>
        <td>string</td>
        <td>
          Minute defines the minute when the policy MUST be granted.
You can also provide a range by filling the minuteEnd field.
In this case, permission is granted only if current minute is between or equal to the two values you provided.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minuteEnd</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>month</b></td>
        <td>string</td>
        <td>
          Month defines the month which the policy MUST be granted.
You can also provide a range by filling the monthEnd.
In this case, permission is granted only if current month is between or equal to the two values you provided.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>monthEnd</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.policies[index].userPolicy
<sup><sup>[↩ Parent](#keycloakclientspecauthorizationpoliciesindex)</sup></sup>



UserPolicy is a user policy settings.

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
        <td><b>users</b></td>
        <td>[]string</td>
        <td>
          Users is a list of usernames. Specifies which user(s) are allowed by this policy.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakClient.spec.authorization.resources[index]
<sup><sup>[↩ Parent](#keycloakclientspecauthorization)</sup></sup>





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
        <td><b>displayName</b></td>
        <td>string</td>
        <td>
          DisplayName for Identity Providers.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is unique resource name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>attributes</b></td>
        <td>map[string][]string</td>
        <td>
          Attributes is a map of resource attributes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>iconUri</b></td>
        <td>string</td>
        <td>
          IconURI pointing to an icon.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ownerManagedAccess</b></td>
        <td>boolean</td>
        <td>
          OwnerManagedAccess if enabled, the access to this resource can be managed by the resource owner.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scopes</b></td>
        <td>[]string</td>
        <td>
          Scopes requested or assigned in advance to the client to determine whether the policy is applied to this client.
Condition is evaluated during OpenID Connect authorization request and/or token request.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type of this resource. It can be used to group different resource instances with the same type.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>uris</b></td>
        <td>[]string</td>
        <td>
          URIs which are protected by resource.<br/>
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


### KeycloakClient.spec.realmRef
<sup><sup>[↩ Parent](#keycloakclientspec)</sup></sup>



RealmRef is reference to Realm custom resource.

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
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind specifies the kind of the Keycloak resource.<br/>
          <br/>
            <i>Enum</i>: KeycloakRealm, ClusterKeycloakRealm<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Keycloak resource.<br/>
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
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Deprecated: use RealmRef instead.
Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakclientscopespecrealmref">realmRef</a></b></td>
        <td>object</td>
        <td>
          RealmRef is reference to Realm custom resource.<br/>
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


### KeycloakClientScope.spec.realmRef
<sup><sup>[↩ Parent](#keycloakclientscopespec)</sup></sup>



RealmRef is reference to Realm custom resource.

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
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind specifies the kind of the Keycloak resource.<br/>
          <br/>
            <i>Enum</i>: KeycloakRealm, ClusterKeycloakRealm<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Keycloak resource.<br/>
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
        <td><b>config</b></td>
        <td>map[string][]string</td>
        <td>
          Config is a map of component configuration.
Map key is a name of configuration property, map value is an array value of configuration properties.
Any configuration property can be a reference to k8s secret, in this case the property should be in format $secretName:secretKey.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmcomponentspecparentref">parentRef</a></b></td>
        <td>object</td>
        <td>
          ParentRef specifies a parent resource.
If not specified, then parent is realm specified in realm field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Deprecated: use RealmRef instead.
Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmcomponentspecrealmref">realmRef</a></b></td>
        <td>object</td>
        <td>
          RealmRef is reference to Realm custom resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealmComponent.spec.parentRef
<sup><sup>[↩ Parent](#keycloakrealmcomponentspec)</sup></sup>



ParentRef specifies a parent resource.
If not specified, then parent is realm specified in realm field.

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
          Name is a name of parent component custom resource.
For example, if Kind is KeycloakRealm, then Name is name of KeycloakRealm custom resource.<br/>
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


### KeycloakRealmComponent.spec.realmRef
<sup><sup>[↩ Parent](#keycloakrealmcomponentspec)</sup></sup>



RealmRef is reference to Realm custom resource.

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
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind specifies the kind of the Keycloak resource.<br/>
          <br/>
            <i>Enum</i>: KeycloakRealm, ClusterKeycloakRealm<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Keycloak resource.<br/>
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
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Deprecated: use RealmRef instead.
Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmgroupspecrealmref">realmRef</a></b></td>
        <td>object</td>
        <td>
          RealmRef is reference to Realm custom resource.<br/>
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


### KeycloakRealmGroup.spec.realmRef
<sup><sup>[↩ Parent](#keycloakrealmgroupspec)</sup></sup>



RealmRef is reference to Realm custom resource.

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
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind specifies the kind of the Keycloak resource.<br/>
          <br/>
            <i>Enum</i>: KeycloakRealm, ClusterKeycloakRealm<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Keycloak resource.<br/>
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
          Config is a map of identity provider configuration.
Map key is a name of configuration property, map value is a value of configuration property.
Any value can be a reference to k8s secret, in this case value should be in format $secretName:secretKey.<br/>
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
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Deprecated: use RealmRef instead.
Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmidentityproviderspecrealmref">realmRef</a></b></td>
        <td>object</td>
        <td>
          RealmRef is reference to Realm custom resource.<br/>
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


### KeycloakRealmIdentityProvider.spec.realmRef
<sup><sup>[↩ Parent](#keycloakrealmidentityproviderspec)</sup></sup>



RealmRef is reference to Realm custom resource.

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
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind specifies the kind of the Keycloak resource.<br/>
          <br/>
            <i>Enum</i>: KeycloakRealm, ClusterKeycloakRealm<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Keycloak resource.<br/>
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
        <td><b><a href="#keycloakrealmrolebatchspecrolesindex">roles</a></b></td>
        <td>[]object</td>
        <td>
          Roles is a list of roles to be created.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Deprecated: use RealmRef instead.
Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolebatchspecrealmref">realmRef</a></b></td>
        <td>object</td>
        <td>
          RealmRef is reference to Realm custom resource.<br/>
        </td>
        <td>false</td>
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


### KeycloakRealmRoleBatch.spec.realmRef
<sup><sup>[↩ Parent](#keycloakrealmrolebatchspec)</sup></sup>



RealmRef is reference to Realm custom resource.

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
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind specifies the kind of the Keycloak resource.<br/>
          <br/>
            <i>Enum</i>: KeycloakRealm, ClusterKeycloakRealm<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Keycloak resource.<br/>
        </td>
        <td>false</td>
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
        <td><b><a href="#keycloakrealmrolespeccompositesclientroleskeyindex">compositesClientRoles</a></b></td>
        <td>map[string][]object</td>
        <td>
          CompositesClientRoles is a map of composites client roles assigned to role.<br/>
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
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Deprecated: use RealmRef instead.
Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmrolespecrealmref">realmRef</a></b></td>
        <td>object</td>
        <td>
          RealmRef is reference to Realm custom resource.<br/>
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


### KeycloakRealmRole.spec.compositesClientRoles[key][index]
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


### KeycloakRealmRole.spec.realmRef
<sup><sup>[↩ Parent](#keycloakrealmrolespec)</sup></sup>



RealmRef is reference to Realm custom resource.

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
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind specifies the kind of the Keycloak resource.<br/>
          <br/>
            <i>Enum</i>: KeycloakRealm, ClusterKeycloakRealm<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Keycloak resource.<br/>
        </td>
        <td>false</td>
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
        <td><b>displayHtmlName</b></td>
        <td>string</td>
        <td>
          DisplayHTMLName name to render in the UI<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>displayName</b></td>
        <td>string</td>
        <td>
          DisplayName is the display name of the realm.<br/>
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
          Deprecated: use KeycloakRef instead.
KeycloakOwner specifies the name of the Keycloak instance that owns the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspeckeycloakref">keycloakRef</a></b></td>
        <td>object</td>
        <td>
          KeycloakRef is reference to Keycloak custom resource.<br/>
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
        <td><b><a href="#keycloakrealmspecsmtp">smtp</a></b></td>
        <td>object</td>
        <td>
          Smtp is the configuration for email in the realm.<br/>
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
        <td><b><a href="#keycloakrealmspectokensettings">tokenSettings</a></b></td>
        <td>object</td>
        <td>
          TokenSettings is the configuration for tokens in the realm.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecuserprofileconfig">userProfileConfig</a></b></td>
        <td>object</td>
        <td>
          UserProfileConfig is the configuration for user profiles in the realm.
Attributes and groups will be added to the current realm configuration.
Deletion of attributes and groups is not supported.<br/>
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


### KeycloakRealm.spec.keycloakRef
<sup><sup>[↩ Parent](#keycloakrealmspec)</sup></sup>



KeycloakRef is reference to Keycloak custom resource.

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
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind specifies the kind of the Keycloak resource.<br/>
          <br/>
            <i>Enum</i>: Keycloak, ClusterKeycloak<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Keycloak resource.<br/>
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


### KeycloakRealm.spec.smtp
<sup><sup>[↩ Parent](#keycloakrealmspec)</sup></sup>



Smtp is the configuration for email in the realm.

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
        <td><b><a href="#keycloakrealmspecsmtpconnection">connection</a></b></td>
        <td>object</td>
        <td>
          Connection specifies the email connection configuration.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecsmtptemplate">template</a></b></td>
        <td>object</td>
        <td>
          Template specifies the email template configuration.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.smtp.connection
<sup><sup>[↩ Parent](#keycloakrealmspecsmtp)</sup></sup>



Connection specifies the email connection configuration.

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
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host specifies the email server host.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecsmtpconnectionauthentication">authentication</a></b></td>
        <td>object</td>
        <td>
          Authentication specifies the email authentication configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableSSL</b></td>
        <td>boolean</td>
        <td>
          EnableSSL specifies if SSL is enabled.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableStartTLS</b></td>
        <td>boolean</td>
        <td>
          EnableStartTLS specifies if StartTLS is enabled.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port specifies the email server port.<br/>
          <br/>
            <i>Default</i>: 25<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.smtp.connection.authentication
<sup><sup>[↩ Parent](#keycloakrealmspecsmtpconnection)</sup></sup>



Authentication specifies the email authentication configuration.

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
        <td><b><a href="#keycloakrealmspecsmtpconnectionauthenticationpassword">password</a></b></td>
        <td>object</td>
        <td>
          Password specifies login password.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecsmtpconnectionauthenticationusername">username</a></b></td>
        <td>object</td>
        <td>
          Username specifies login username.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.smtp.connection.authentication.password
<sup><sup>[↩ Parent](#keycloakrealmspecsmtpconnectionauthentication)</sup></sup>



Password specifies login password.

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
        <td><b><a href="#keycloakrealmspecsmtpconnectionauthenticationpasswordconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecsmtpconnectionauthenticationpasswordsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.smtp.connection.authentication.password.configMapKeyRef
<sup><sup>[↩ Parent](#keycloakrealmspecsmtpconnectionauthenticationpassword)</sup></sup>



Selects a key of a ConfigMap.

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
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.smtp.connection.authentication.password.secretKeyRef
<sup><sup>[↩ Parent](#keycloakrealmspecsmtpconnectionauthenticationpassword)</sup></sup>



Selects a key of a secret.

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
          The key of the secret to select from.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.smtp.connection.authentication.username
<sup><sup>[↩ Parent](#keycloakrealmspecsmtpconnectionauthentication)</sup></sup>



Username specifies login username.

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
        <td><b><a href="#keycloakrealmspecsmtpconnectionauthenticationusernameconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecsmtpconnectionauthenticationusernamesecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Directly specifies a value.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.smtp.connection.authentication.username.configMapKeyRef
<sup><sup>[↩ Parent](#keycloakrealmspecsmtpconnectionauthenticationusername)</sup></sup>



Selects a key of a ConfigMap.

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
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.smtp.connection.authentication.username.secretKeyRef
<sup><sup>[↩ Parent](#keycloakrealmspecsmtpconnectionauthenticationusername)</sup></sup>



Selects a key of a secret.

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
          The key of the secret to select from.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.smtp.template
<sup><sup>[↩ Parent](#keycloakrealmspecsmtp)</sup></sup>



Template specifies the email template configuration.

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
        <td><b>from</b></td>
        <td>string</td>
        <td>
          From specifies the sender email address.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>envelopeFrom</b></td>
        <td>string</td>
        <td>
          EnvelopeFrom is an email address used for bounces .<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>fromDisplayName</b></td>
        <td>string</td>
        <td>
          FromDisplayName specifies the sender display for sender email address.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replyTo</b></td>
        <td>string</td>
        <td>
          ReplyTo specifies the reply-to email address.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replyToDisplayName</b></td>
        <td>string</td>
        <td>
          ReplyToDisplayName specifies display name for reply-to email address.<br/>
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


### KeycloakRealm.spec.tokenSettings
<sup><sup>[↩ Parent](#keycloakrealmspec)</sup></sup>



TokenSettings is the configuration for tokens in the realm.

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
        <td><b>accessCodeLifespan</b></td>
        <td>integer</td>
        <td>
          AccessCodeLifespan specifies max time(in seconds)a client has to finish the access token protocol.
This should normally be 1 minute.<br/>
          <br/>
            <i>Default</i>: 60<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>accessToken</b></td>
        <td>integer</td>
        <td>
          AccessTokenLifespanForImplicitFlow specifies max time(in seconds) before an access token is expired for implicit flow.<br/>
          <br/>
            <i>Default</i>: 900<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>accessTokenLifespan</b></td>
        <td>integer</td>
        <td>
          AccessTokenLifespan specifies max time(in seconds) before an access token is expired.
This value is recommended to be short relative to the SSO timeout.<br/>
          <br/>
            <i>Default</i>: 300<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>actionTokenGeneratedByAdminLifespan</b></td>
        <td>integer</td>
        <td>
          ActionTokenGeneratedByAdminLifespan specifies max time(in seconds) before an action permit sent to a user by administrator is expired.
This value is recommended to be long to allow administrators to send e-mails for users that are currently offline.
The default timeout can be overridden immediately before issuing the token.<br/>
          <br/>
            <i>Default</i>: 43200<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>actionTokenGeneratedByUserLifespan</b></td>
        <td>integer</td>
        <td>
          AccessCodeLifespanUserAction specifies max time(in seconds) before an action permit sent by a user (such as a forgot password e-mail) is expired.
This value is recommended to be short because it's expected that the user would react to self-created action quickly.<br/>
          <br/>
            <i>Default</i>: 300<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>defaultSignatureAlgorithm</b></td>
        <td>enum</td>
        <td>
          DefaultSignatureAlgorithm specifies the default algorithm used to sign tokens for the realm<br/>
          <br/>
            <i>Enum</i>: ES256, ES384, ES512, EdDSA, HS256, HS384, HS512, PS256, PS384, PS512, RS256, RS384, RS512<br/>
            <i>Default</i>: RS256<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>refreshTokenMaxReuse</b></td>
        <td>integer</td>
        <td>
          RefreshTokenMaxReuse specifies maximum number of times a refresh token can be reused.
When a different token is used, revocation is immediate.<br/>
          <br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>revokeRefreshToken</b></td>
        <td>boolean</td>
        <td>
          RevokeRefreshToken if enabled a refresh token can only be used up to 'refreshTokenMaxReuse' and
is revoked when a different token is used.
Otherwise, refresh tokens are not revoked when used and can be used multiple times.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.userProfileConfig
<sup><sup>[↩ Parent](#keycloakrealmspec)</sup></sup>



UserProfileConfig is the configuration for user profiles in the realm.
Attributes and groups will be added to the current realm configuration.
Deletion of attributes and groups is not supported.

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
        <td><b><a href="#keycloakrealmspecuserprofileconfigattributesindex">attributes</a></b></td>
        <td>[]object</td>
        <td>
          Attributes specifies the list of user profile attributes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecuserprofileconfiggroupsindex">groups</a></b></td>
        <td>[]object</td>
        <td>
          Groups specifies the list of user profile groups.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>unmanagedAttributePolicy</b></td>
        <td>string</td>
        <td>
          UnmanagedAttributePolicy are user attributes not explicitly defined in the user profile configuration.
Empty value means that unmanaged attributes are disabled.
Possible values:
ENABLED - unmanaged attributes are allowed.
ADMIN_VIEW - unmanaged attributes are read-only and only available through the administration console and API.
ADMIN_EDIT - unmanaged attributes can be managed only through the administration console and API.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.userProfileConfig.attributes[index]
<sup><sup>[↩ Parent](#keycloakrealmspecuserprofileconfig)</sup></sup>





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
          Name of the user attribute, used to uniquely identify an attribute.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td>
          Annotations specifies the annotations for the attribute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>displayName</b></td>
        <td>string</td>
        <td>
          Display name for the attribute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>group</b></td>
        <td>string</td>
        <td>
          Group to which the attribute belongs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>multivalued</b></td>
        <td>boolean</td>
        <td>
          Multivalued specifies if this attribute supports multiple values.
This setting is an indicator and does not enable any validation<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecuserprofileconfigattributesindexpermissions">permissions</a></b></td>
        <td>object</td>
        <td>
          Permissions specifies the permissions for the attribute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecuserprofileconfigattributesindexrequired">required</a></b></td>
        <td>object</td>
        <td>
          Required indicates that the attribute must be set by users and administrators.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecuserprofileconfigattributesindexselector">selector</a></b></td>
        <td>object</td>
        <td>
          Selector specifies the scopes for which the attribute is available.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmspecuserprofileconfigattributesindexvalidationskeykey">validations</a></b></td>
        <td>map[string]map[string]object</td>
        <td>
          Validations specifies the validations for the attribute.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.userProfileConfig.attributes[index].permissions
<sup><sup>[↩ Parent](#keycloakrealmspecuserprofileconfigattributesindex)</sup></sup>



Permissions specifies the permissions for the attribute.

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
        <td><b>edit</b></td>
        <td>[]string</td>
        <td>
          Edit specifies who can edit the attribute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>view</b></td>
        <td>[]string</td>
        <td>
          View specifies who can view the attribute.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.userProfileConfig.attributes[index].required
<sup><sup>[↩ Parent](#keycloakrealmspecuserprofileconfigattributesindex)</sup></sup>



Required indicates that the attribute must be set by users and administrators.

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
        <td><b>roles</b></td>
        <td>[]string</td>
        <td>
          Roles specifies the roles for whom the attribute is required.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>scopes</b></td>
        <td>[]string</td>
        <td>
          Scopes specifies the scopes when the attribute is required.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.userProfileConfig.attributes[index].selector
<sup><sup>[↩ Parent](#keycloakrealmspecuserprofileconfigattributesindex)</sup></sup>



Selector specifies the scopes for which the attribute is available.

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
        <td><b>scopes</b></td>
        <td>[]string</td>
        <td>
          Scopes specifies the scopes for which the attribute is available.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.userProfileConfig.attributes[index].validations[key][key]
<sup><sup>[↩ Parent](#keycloakrealmspecuserprofileconfigattributesindex)</sup></sup>





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
        <td><b>intVal</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mapVal</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sliceVal</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>stringVal</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### KeycloakRealm.spec.userProfileConfig.groups[index]
<sup><sup>[↩ Parent](#keycloakrealmspecuserprofileconfig)</sup></sup>





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
          Name is unique name of the group.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td>
          Annotations specifies the annotations for the group.
nullable<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>displayDescription</b></td>
        <td>string</td>
        <td>
          DisplayDescription specifies a user-friendly name for the group that should be used when rendering a group of attributes in user-facing forms.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>displayHeader</b></td>
        <td>string</td>
        <td>
          DisplayHeader specifies a text that should be used as a header when rendering user-facing forms.<br/>
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
          KeepResource, when set to false, results in the deletion of the KeycloakRealmUser Custom Resource (CR)
from the cluster after the corresponding user is created in Keycloak. The user will continue to exist in Keycloak.
When set to true, the CR will not be deleted after processing.<br/>
          <br/>
            <i>Default</i>: true<br/>
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
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          Deprecated: use RealmRef instead.
Realm is name of KeycloakRealm custom resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakrealmuserspecrealmref">realmRef</a></b></td>
        <td>object</td>
        <td>
          RealmRef is reference to Realm custom resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reconciliationStrategy</b></td>
        <td>string</td>
        <td>
          ReconciliationStrategy is a strategy for reconciliation. Possible values: full, create-only.
Default value: full. If set to create-only, user will be created only if it does not exist. If user exists, it will not be updated.
If set to full, user will be created if it does not exist, or updated if it exists.<br/>
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


### KeycloakRealmUser.spec.realmRef
<sup><sup>[↩ Parent](#keycloakrealmuserspec)</sup></sup>



RealmRef is reference to Realm custom resource.

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
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          Kind specifies the kind of the Keycloak resource.<br/>
          <br/>
            <i>Enum</i>: KeycloakRealm, ClusterKeycloakRealm<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Keycloak resource.<br/>
        </td>
        <td>false</td>
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
          <br/>
            <i>Default</i>: map[connected:false]<br/>
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
      </tr><tr>
        <td><b><a href="#keycloakspeccacert">caCert</a></b></td>
        <td>object</td>
        <td>
          CACert defines the root certificate authority
that api client use when verifying server certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          InsecureSkipVerify controls whether api client verifies the server's
certificate chain and host name. If InsecureSkipVerify is true, api client
accepts any certificate presented by the server and any host name in that
certificate.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Keycloak.spec.caCert
<sup><sup>[↩ Parent](#keycloakspec)</sup></sup>



CACert defines the root certificate authority
that api client use when verifying server certificates.

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
        <td><b><a href="#keycloakspeccacertconfigmapkeyref">configMapKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a ConfigMap.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#keycloakspeccacertsecretkeyref">secretKeyRef</a></b></td>
        <td>object</td>
        <td>
          Selects a key of a secret.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Keycloak.spec.caCert.configMapKeyRef
<sup><sup>[↩ Parent](#keycloakspeccacert)</sup></sup>



Selects a key of a ConfigMap.

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
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Keycloak.spec.caCert.secretKeyRef
<sup><sup>[↩ Parent](#keycloakspeccacert)</sup></sup>



Selects a key of a secret.

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
          The key of the secret to select from.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
TODO: Add other useful fields. apiVersion, kind, uid?<br/>
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