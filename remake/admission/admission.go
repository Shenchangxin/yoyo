// Copyright 2026 Hypit.AI
// Translated from @hypit/admission (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package admission

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/remake/facet"
	"github.com/Shenchangxin/yoyo/remake/kernel"
	"github.com/Shenchangxin/yoyo/remake/protocol"
)

// AdmissionPackageFacetABI is the facet ABI for package-owned semantic validators.
const AdmissionPackageFacetABI = "hypit.admission-package@1"

// TypeValidatorContext is the value handed to a semantic validator.
// Type and Value are copies; a handler cannot change the record being admitted.
type TypeValidatorContext struct {
	Type  protocol.TypeRef
	Value protocol.StoredValue
}

// TypeValidatorHandler throws to reject a value.
type TypeValidatorHandler func(TypeValidatorContext)

// TypeValidatorRegistrar accepts one handler per type.
type TypeValidatorRegistrar interface {
	Register(protocol.TypeRef, TypeValidatorHandler)
}

// TypeValidatorFacet is one package-owned semantic validator.
type TypeValidatorFacet struct {
	Type    protocol.TypeRef
	Handler TypeValidatorHandler
}

// AdmissionPackage is the implementation of an admission facet.
// A nil Validators slice means the field was absent.
type AdmissionPackage struct {
	Validators []TypeValidatorFacet
}

// AdmissionPackageFacet is a facet whose ABI is AdmissionPackageFacetABI.
type AdmissionPackageFacet struct {
	ABI            string
	Implementation AdmissionPackage
}

// CreateAdmissionPackageFacet keeps validators only when the value is an object that has them.
func CreateAdmissionPackageFacet(value any) AdmissionPackageFacet {
	item, ok := value.(AdmissionPackage)
	if !ok || item.Validators == nil {
		return AdmissionPackageFacet{ABI: AdmissionPackageFacetABI}
	}
	return AdmissionPackageFacet{
		ABI:            AdmissionPackageFacetABI,
		Implementation: AdmissionPackage{Validators: item.Validators},
	}
}

// AdmissionPackagesFromFacets returns implementations whose ABI is the admission package ABI.
func AdmissionPackagesFromFacets(facets []facet.Facet) []AdmissionPackage {
	var out []AdmissionPackage
	for _, item := range facets {
		if item.ABI != AdmissionPackageFacetABI {
			continue
		}
		impl, ok := item.Implementation.(AdmissionPackage)
		if !ok {
			continue
		}
		out = append(out, impl)
	}
	return out
}

// RegisterTypeValidatorFacets registers each facet on the registry.
func RegisterTypeValidatorFacets(registry TypeValidatorRegistrar, facets []TypeValidatorFacet) {
	for _, item := range facets {
		registry.Register(item.Type, item.Handler)
	}
}

// TypeValidatorRegistration is the handler stored for one type.
type TypeValidatorRegistration struct {
	Handler TypeValidatorHandler
}

// TypeValidatorRegistryLike resolves a handler for a type.
type TypeValidatorRegistryLike interface {
	Resolve(protocol.TypeRef) (TypeValidatorRegistration, bool)
}

// TypeValidationError is the failure thrown by registration and refinement.
type TypeValidationError struct {
	Code    string
	Message string
	Subject *string
}

func (e *TypeValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// TypeValidatorRegistry stores one handler per type key.
type TypeValidatorRegistry struct {
	validators map[string]TypeValidatorRegistration
}

func (r *TypeValidatorRegistry) Register(ref protocol.TypeRef, handler TypeValidatorHandler) {
	if r.validators == nil {
		r.validators = map[string]TypeValidatorRegistration{}
	}
	key := typeKey(ref)
	if _, exists := r.validators[key]; exists {
		panic(&TypeValidationError{
			Code:    "DUPLICATE_TYPE_VALIDATOR",
			Message: key + " validator is already registered",
			Subject: &key,
		})
	}
	r.validators[key] = TypeValidatorRegistration{Handler: handler}
}

func (r *TypeValidatorRegistry) Resolve(ref protocol.TypeRef) (TypeValidatorRegistration, bool) {
	if r == nil || r.validators == nil {
		return TypeValidatorRegistration{}, false
	}
	registration, ok := r.validators[typeKey(ref)]
	return registration, ok
}

func refineValue(ref protocol.TypeRef, value protocol.StoredValue, registry TypeValidatorRegistryLike) {
	registration, ok := registry.Resolve(ref)
	if !ok {
		return
	}
	defer func() {
		rec := recover()
		if rec == nil {
			return
		}
		key := typeKey(ref)
		panic(&TypeValidationError{
			Code:    "TYPE_REFINEMENT_REJECTED",
			Message: key + " rejected the value: " + panicText(rec),
			Subject: &key,
		})
	}()
	registration.Handler(TypeValidatorContext{Type: ref, Value: cloneStored(value)})
}

// ValidateValue resolves the type, then runs its semantic handler when one is registered.
func ValidateValue(closure protocol.ResolvedModuleClosure, ref protocol.TypeRef, value protocol.StoredValue, registry TypeValidatorRegistryLike) {
	kernel.ResolveType(closure, ref)
	refineValue(ref, value, registry)
}

// AdmitRecord checks the record structure and its semantic handler, then returns that record.
func AdmitRecord(closure protocol.ResolvedModuleClosure, record protocol.TypedRecord, registry TypeValidatorRegistryLike) protocol.TypedRecord {
	kernel.VerifyRecordStructure(closure, record)
	refineValue(record.Type, record.Value, registry)
	return record
}

// CreateRecordAdmitter binds a registry to the admission function used by a host.
func CreateRecordAdmitter(registry TypeValidatorRegistryLike) func(protocol.ResolvedModuleClosure, protocol.TypedRecord) {
	return func(closure protocol.ResolvedModuleClosure, record protocol.TypedRecord) {
		AdmitRecord(closure, record, registry)
	}
}

func typeKey(ref protocol.TypeRef) string {
	return ref.Module.Name + "@" + ref.Module.Version + "#" + ref.Name
}

func panicText(value any) string {
	switch item := value.(type) {
	case error:
		return item.Error()
	default:
		return fmt.Sprint(item)
	}
}

func cloneStored(value protocol.StoredValue) protocol.StoredValue {
	out := value
	out.Value = cloneCanonical(value.Value)
	return out
}

func cloneCanonical(value any) any {
	switch item := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(item))
		for key, child := range item {
			out[key] = cloneCanonical(child)
		}
		return out
	case []any:
		out := make([]any, len(item))
		for i, child := range item {
			out[i] = cloneCanonical(child)
		}
		return out
	default:
		return value
	}
}
