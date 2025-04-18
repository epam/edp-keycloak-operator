//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package common

import ()

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigMapKeySelector) DeepCopyInto(out *ConfigMapKeySelector) {
	*out = *in
	out.LocalObjectReference = in.LocalObjectReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigMapKeySelector.
func (in *ConfigMapKeySelector) DeepCopy() *ConfigMapKeySelector {
	if in == nil {
		return nil
	}
	out := new(ConfigMapKeySelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmailAuthentication) DeepCopyInto(out *EmailAuthentication) {
	*out = *in
	in.Username.DeepCopyInto(&out.Username)
	in.Password.DeepCopyInto(&out.Password)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmailAuthentication.
func (in *EmailAuthentication) DeepCopy() *EmailAuthentication {
	if in == nil {
		return nil
	}
	out := new(EmailAuthentication)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmailConnection) DeepCopyInto(out *EmailConnection) {
	*out = *in
	if in.Authentication != nil {
		in, out := &in.Authentication, &out.Authentication
		*out = new(EmailAuthentication)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmailConnection.
func (in *EmailConnection) DeepCopy() *EmailConnection {
	if in == nil {
		return nil
	}
	out := new(EmailConnection)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmailTemplate) DeepCopyInto(out *EmailTemplate) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmailTemplate.
func (in *EmailTemplate) DeepCopy() *EmailTemplate {
	if in == nil {
		return nil
	}
	out := new(EmailTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KeycloakRef) DeepCopyInto(out *KeycloakRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KeycloakRef.
func (in *KeycloakRef) DeepCopy() *KeycloakRef {
	if in == nil {
		return nil
	}
	out := new(KeycloakRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RealmRef) DeepCopyInto(out *RealmRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RealmRef.
func (in *RealmRef) DeepCopy() *RealmRef {
	if in == nil {
		return nil
	}
	out := new(RealmRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SMTP) DeepCopyInto(out *SMTP) {
	*out = *in
	out.Template = in.Template
	in.Connection.DeepCopyInto(&out.Connection)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SMTP.
func (in *SMTP) DeepCopy() *SMTP {
	if in == nil {
		return nil
	}
	out := new(SMTP)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretKeySelector) DeepCopyInto(out *SecretKeySelector) {
	*out = *in
	out.LocalObjectReference = in.LocalObjectReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretKeySelector.
func (in *SecretKeySelector) DeepCopy() *SecretKeySelector {
	if in == nil {
		return nil
	}
	out := new(SecretKeySelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SourceRef) DeepCopyInto(out *SourceRef) {
	*out = *in
	if in.ConfigMapKeyRef != nil {
		in, out := &in.ConfigMapKeyRef, &out.ConfigMapKeyRef
		*out = new(ConfigMapKeySelector)
		**out = **in
	}
	if in.SecretKeyRef != nil {
		in, out := &in.SecretKeyRef, &out.SecretKeyRef
		*out = new(SecretKeySelector)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SourceRef.
func (in *SourceRef) DeepCopy() *SourceRef {
	if in == nil {
		return nil
	}
	out := new(SourceRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SourceRefOrVal) DeepCopyInto(out *SourceRefOrVal) {
	*out = *in
	if in.SourceRef != nil {
		in, out := &in.SourceRef, &out.SourceRef
		*out = new(SourceRef)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SourceRefOrVal.
func (in *SourceRefOrVal) DeepCopy() *SourceRefOrVal {
	if in == nil {
		return nil
	}
	out := new(SourceRefOrVal)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenSettings) DeepCopyInto(out *TokenSettings) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenSettings.
func (in *TokenSettings) DeepCopy() *TokenSettings {
	if in == nil {
		return nil
	}
	out := new(TokenSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserProfileAttribute) DeepCopyInto(out *UserProfileAttribute) {
	*out = *in
	if in.Permissions != nil {
		in, out := &in.Permissions, &out.Permissions
		*out = new(UserProfileAttributePermissions)
		(*in).DeepCopyInto(*out)
	}
	if in.Required != nil {
		in, out := &in.Required, &out.Required
		*out = new(UserProfileAttributeRequired)
		(*in).DeepCopyInto(*out)
	}
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(UserProfileAttributeSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Validations != nil {
		in, out := &in.Validations, &out.Validations
		*out = make(map[string]map[string]UserProfileAttributeValidation, len(*in))
		for key, val := range *in {
			var outVal map[string]UserProfileAttributeValidation
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make(map[string]UserProfileAttributeValidation, len(*in))
				for key, val := range *in {
					(*out)[key] = *val.DeepCopy()
				}
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserProfileAttribute.
func (in *UserProfileAttribute) DeepCopy() *UserProfileAttribute {
	if in == nil {
		return nil
	}
	out := new(UserProfileAttribute)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserProfileAttributePermissions) DeepCopyInto(out *UserProfileAttributePermissions) {
	*out = *in
	if in.Edit != nil {
		in, out := &in.Edit, &out.Edit
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.View != nil {
		in, out := &in.View, &out.View
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserProfileAttributePermissions.
func (in *UserProfileAttributePermissions) DeepCopy() *UserProfileAttributePermissions {
	if in == nil {
		return nil
	}
	out := new(UserProfileAttributePermissions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserProfileAttributeRequired) DeepCopyInto(out *UserProfileAttributeRequired) {
	*out = *in
	if in.Roles != nil {
		in, out := &in.Roles, &out.Roles
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Scopes != nil {
		in, out := &in.Scopes, &out.Scopes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserProfileAttributeRequired.
func (in *UserProfileAttributeRequired) DeepCopy() *UserProfileAttributeRequired {
	if in == nil {
		return nil
	}
	out := new(UserProfileAttributeRequired)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserProfileAttributeSelector) DeepCopyInto(out *UserProfileAttributeSelector) {
	*out = *in
	if in.Scopes != nil {
		in, out := &in.Scopes, &out.Scopes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserProfileAttributeSelector.
func (in *UserProfileAttributeSelector) DeepCopy() *UserProfileAttributeSelector {
	if in == nil {
		return nil
	}
	out := new(UserProfileAttributeSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserProfileAttributeValidation) DeepCopyInto(out *UserProfileAttributeValidation) {
	*out = *in
	if in.MapVal != nil {
		in, out := &in.MapVal, &out.MapVal
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.SliceVal != nil {
		in, out := &in.SliceVal, &out.SliceVal
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserProfileAttributeValidation.
func (in *UserProfileAttributeValidation) DeepCopy() *UserProfileAttributeValidation {
	if in == nil {
		return nil
	}
	out := new(UserProfileAttributeValidation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserProfileConfig) DeepCopyInto(out *UserProfileConfig) {
	*out = *in
	if in.Attributes != nil {
		in, out := &in.Attributes, &out.Attributes
		*out = make([]UserProfileAttribute, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Groups != nil {
		in, out := &in.Groups, &out.Groups
		*out = make([]UserProfileGroup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserProfileConfig.
func (in *UserProfileConfig) DeepCopy() *UserProfileConfig {
	if in == nil {
		return nil
	}
	out := new(UserProfileConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserProfileGroup) DeepCopyInto(out *UserProfileGroup) {
	*out = *in
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserProfileGroup.
func (in *UserProfileGroup) DeepCopy() *UserProfileGroup {
	if in == nil {
		return nil
	}
	out := new(UserProfileGroup)
	in.DeepCopyInto(out)
	return out
}
