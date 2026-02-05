package chain

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type UserProfile struct {
	next handler.RealmHandler
}

func (a UserProfile) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client, kClientV2 *keycloakv2.KeycloakClient) error {
	l := ctrl.LoggerFrom(ctx)

	if realm.Spec.UserProfileConfig == nil {
		l.Info("User profile is empty, skipping configuration")

		return nextServeOrNil(ctx, a.next, realm, kClient, kClientV2)
	}

	l.Info("Start configuring keycloak realm user profile")

	err := ProcessUserProfile(ctx, realm.Spec.RealmName, realm.Spec.UserProfileConfig, kClientV2)
	if err != nil {
		return err
	}

	l.Info("User profile has been configured")

	return nextServeOrNil(ctx, a.next, realm, kClient, kClientV2)
}

func ProcessUserProfile(ctx context.Context, realm string, userProfileSpec *common.UserProfileConfig, kClientV2 *keycloakv2.KeycloakClient) error {
	userProfile, _, err := kClientV2.Users.GetUsersProfile(ctx, realm)
	if err != nil {
		return fmt.Errorf("unable to get current user profile: %w", err)
	}

	userProfileToUpdate := userProfileConfigSpecToModel(userProfileSpec)
	attributesToUpdate := userProfileConfigAttributeToMap(&userProfileToUpdate)

	if userProfile.Attributes == nil {
		userProfile.Attributes = &[]keycloakv2.UserProfileAttribute{}
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
		userProfile.Groups = &[]keycloakv2.UserProfileGroup{}
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

	if _, _, err = kClientV2.Users.UpdateUsersProfile(
		ctx,
		realm,
		*userProfile,
	); err != nil {
		return fmt.Errorf("unable to update user profile: %w", err)
	}

	return nil
}

func userProfileConfigAttributeToMap(profile *keycloakv2.UserProfileConfig) map[string]keycloakv2.UserProfileAttribute {
	if profile.Attributes == nil {
		return make(map[string]keycloakv2.UserProfileAttribute)
	}

	attributes := make(map[string]keycloakv2.UserProfileAttribute, len(*profile.Attributes))

	for _, v := range *profile.Attributes {
		attributes[*v.Name] = v
	}

	return attributes
}

func userProfileConfigGroupToMap(spec *keycloakv2.UserProfileConfig) map[string]keycloakv2.UserProfileGroup {
	if spec.Groups == nil {
		return make(map[string]keycloakv2.UserProfileGroup)
	}

	groups := make(map[string]keycloakv2.UserProfileGroup, len(*spec.Groups))

	for _, v := range *spec.Groups {
		groups[*v.Name] = v
	}

	return groups
}

func userProfileConfigSpecToModel(spec *common.UserProfileConfig) keycloakv2.UserProfileConfig {
	userProfile := keycloakv2.UserProfileConfig{}

	if spec.UnmanagedAttributePolicy != "" {
		userProfile.UnmanagedAttributePolicy = ptr.To(keycloakv2.UnmanagedAttributePolicy(spec.UnmanagedAttributePolicy))
	}

	if spec.Attributes != nil {
		attributes := make([]keycloakv2.UserProfileAttribute, 0, len(spec.Attributes))

		for _, v := range spec.Attributes {
			attr := userProfileConfigAttributeSpecToModel(&v)

			attributes = append(attributes, attr)
		}

		userProfile.Attributes = &attributes
	}

	if spec.Groups != nil {
		groups := make([]keycloakv2.UserProfileGroup, 0, len(spec.Groups))

		for _, v := range spec.Groups {
			group := userProfileConfigGroupSpecToModel(v)

			groups = append(groups, group)
		}

		userProfile.Groups = &groups
	}

	return userProfile
}

func userProfileConfigGroupSpecToModel(v common.UserProfileGroup) keycloakv2.UserProfileGroup {
	group := keycloakv2.UserProfileGroup{
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

func userProfileConfigAttributeSpecToModel(v *common.UserProfileAttribute) keycloakv2.UserProfileAttribute {
	if v == nil {
		return keycloakv2.UserProfileAttribute{}
	}

	attr := keycloakv2.UserProfileAttribute{
		DisplayName: &v.DisplayName,
		Name:        &v.Name,
		Multivalued: &v.Multivalued,
	}

	if v.Group != "" {
		attr.Group = &v.Group
	}

	annotations := make(map[string]interface{}, len(v.Annotations))
	for ak, av := range v.Annotations {
		annotations[ak] = av
	}

	attr.Annotations = &annotations
	validations := userProfileConfigValidationSpecToModel(v.Validations)
	attr.Validations = &validations

	if v.Permissions != nil {
		permissions := keycloakv2.UserProfileAttributePermissions{}
		edit := slices.Clone(v.Permissions.Edit)
		permissions.Edit = &edit

		view := slices.Clone(v.Permissions.View)
		permissions.View = &view

		attr.Permissions = &permissions
	}

	if v.Required != nil {
		required := keycloakv2.UserProfileAttributeRequired{}
		roles := slices.Clone(v.Required.Roles)
		required.Roles = &roles

		scopes := slices.Clone(v.Required.Scopes)
		required.Scopes = &scopes

		attr.Required = &required
	}

	if v.Selector != nil {
		selector := keycloakv2.UserProfileAttributeSelector{}
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
