package chain

import (
	"context"
	"fmt"
	"slices"

	keycloakgoclient "github.com/zmotso/keycloak-go-client"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/controllers/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type UserProfile struct {
	next handler.RealmHandler
}

func (a UserProfile) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	l := ctrl.LoggerFrom(ctx)

	if realm.Spec.UserProfileConfig == nil {
		l.Info("User profile is empty, skipping configuration")

		return nextServeOrNil(ctx, a.next, realm, kClient)
	}

	l.Info("Start configuring keycloak realm user profile")

	err := ProcessUserProfile(ctx, realm.Spec.RealmName, realm.Spec.UserProfileConfig, kClient)
	if err != nil {
		return err
	}

	l.Info("User profile has been configured")

	return nextServeOrNil(ctx, a.next, realm, kClient)
}

func ProcessUserProfile(ctx context.Context, realm string, userProfileSpec *common.UserProfileConfig, kClient keycloak.Client) error {
	userProfile, err := kClient.GetUsersProfile(ctx, realm)
	if err != nil {
		return fmt.Errorf("unable to get current user profile: %w", err)
	}

	userProfileToUpdate := userProfileConfigSpecToModel(userProfileSpec)
	attributesToUpdate := userProfileConfigAttributeToMap(&userProfileToUpdate)

	if userProfile.Attributes == nil {
		userProfile.Attributes = &[]keycloakgoclient.UserProfileAttribute{}
	}

	for i := 0; i < len(*userProfile.Attributes); i++ {
		attribute := (*userProfile.Attributes)[i]
		if v, ok := attributesToUpdate[*attribute.Name]; ok {
			(*userProfile.Attributes)[i] = v

			delete(attributesToUpdate, *attribute.Name)
		}
	}

	for _, v := range attributesToUpdate {
		*userProfile.Attributes = append(*userProfile.Attributes, v)
	}

	groupsToUpdate := userProfileConfigGroupToMap(&userProfileToUpdate)

	if userProfile.Groups == nil {
		userProfile.Groups = &[]keycloakgoclient.UserProfileGroup{}
	}

	for i := 0; i < len(*userProfile.Groups); i++ {
		group := (*userProfile.Groups)[i]
		if v, ok := groupsToUpdate[*group.Name]; ok {
			(*userProfile.Groups)[i] = v

			delete(groupsToUpdate, *group.Name)
		}
	}

	for _, v := range groupsToUpdate {
		*userProfile.Groups = append(*userProfile.Groups, v)
	}

	userProfile.UnmanagedAttributePolicy = userProfileToUpdate.UnmanagedAttributePolicy

	if _, err = kClient.UpdateUsersProfile(
		ctx,
		realm,
		*userProfile,
	); err != nil {
		return fmt.Errorf("unable to update user profile: %w", err)
	}

	return nil
}

func userProfileConfigAttributeToMap(profile *keycloakgoclient.UserProfileConfig) map[string]keycloakgoclient.UserProfileAttribute {
	if profile.Attributes == nil {
		return make(map[string]keycloakgoclient.UserProfileAttribute)
	}

	attributes := make(map[string]keycloakgoclient.UserProfileAttribute, len(*profile.Attributes))

	for _, v := range *profile.Attributes {
		attributes[*v.Name] = v
	}

	return attributes
}

func userProfileConfigGroupToMap(spec *keycloakgoclient.UserProfileConfig) map[string]keycloakgoclient.UserProfileGroup {
	if spec.Groups == nil {
		return make(map[string]keycloakgoclient.UserProfileGroup)
	}

	groups := make(map[string]keycloakgoclient.UserProfileGroup, len(*spec.Groups))

	for _, v := range *spec.Groups {
		groups[*v.Name] = v
	}

	return groups
}

func userProfileConfigSpecToModel(spec *common.UserProfileConfig) keycloakgoclient.UserProfileConfig {
	userProfile := keycloakgoclient.UserProfileConfig{}

	if spec.UnmanagedAttributePolicy != "" {
		userProfile.UnmanagedAttributePolicy = ptr.To(keycloakgoclient.UnmanagedAttributePolicy(spec.UnmanagedAttributePolicy))
	}

	if spec.Attributes != nil {
		attributes := make([]keycloakgoclient.UserProfileAttribute, 0, len(spec.Attributes))

		for _, v := range spec.Attributes {
			attr := userProfileConfigAttributeSpecToModel(&v)

			attributes = append(attributes, attr)
		}

		userProfile.Attributes = &attributes
	}

	if spec.Groups != nil {
		groups := make([]keycloakgoclient.UserProfileGroup, 0, len(spec.Groups))

		for _, v := range spec.Groups {
			group := userProfileConfigGroupSpecToModel(v)

			groups = append(groups, group)
		}

		userProfile.Groups = &groups
	}

	return userProfile
}

func userProfileConfigGroupSpecToModel(v common.UserProfileGroup) keycloakgoclient.UserProfileGroup {
	group := keycloakgoclient.UserProfileGroup{
		DisplayDescription: &v.DisplayDescription,
		DisplayHeader:      &v.DisplayHeader,
		Name:               &v.Name,
	}

	annotations := make(map[string]interface{}, len(v.Annotations))
	for ak, av := range v.Annotations {
		annotations[ak] = av
	}

	group.Annotations = &annotations

	return group
}

func userProfileConfigAttributeSpecToModel(v *common.UserProfileAttribute) keycloakgoclient.UserProfileAttribute {
	if v == nil {
		return keycloakgoclient.UserProfileAttribute{}
	}

	attr := keycloakgoclient.UserProfileAttribute{
		DisplayName: &v.DisplayName,
		Group:       &v.Group,
		Name:        &v.Name,
		Multivalued: &v.Multivalued,
	}

	annotations := make(map[string]interface{}, len(v.Annotations))
	for ak, av := range v.Annotations {
		annotations[ak] = av
	}

	attr.Annotations = &annotations
	validations := userProfileConfigValidationSpecToModel(v.Validations)
	attr.Validations = &validations

	if v.Permissions != nil {
		permissions := keycloakgoclient.UserProfileAttributePermissions{}
		edit := slices.Clone(v.Permissions.Edit)
		permissions.Edit = &edit

		view := slices.Clone(v.Permissions.View)
		permissions.View = &view

		attr.Permissions = &permissions
	}

	if v.Required != nil {
		required := keycloakgoclient.UserProfileAttributeRequired{}
		roles := slices.Clone(v.Required.Roles)
		required.Roles = &roles

		scopes := slices.Clone(v.Required.Scopes)
		required.Scopes = &scopes

		attr.Required = &required
	}

	if v.Selector != nil {
		selector := keycloakgoclient.UserProfileAttributeSelector{}
		scopes := slices.Clone(v.Selector.Scopes)
		selector.Scopes = &scopes

		attr.Selector = &selector
	}

	return attr
}

func userProfileConfigValidationSpecToModel(validations map[string]map[string]common.UserProfileAttributeValidation) map[string]map[string]interface{} {
	model := make(map[string]map[string]interface{}, len(validations))

	for validatorName, validatorVal := range validations {
		val := make(map[string]interface{}, len(validatorVal))

		for k, v := range validatorVal {
			if v.StringVal != "" {
				val[k] = v.StringVal
				continue
			}

			if v.MapVal != nil {
				val[k] = v.MapVal
				continue
			}

			if v.IntVal != 0 {
				val[k] = v.IntVal
				continue
			}

			if v.SliceVal != nil {
				val[k] = v.SliceVal
			}
		}

		model[validatorName] = val
	}

	return model
}
