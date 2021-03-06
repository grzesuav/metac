/*
Copyright 2019 The MayaData Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	metacontrollerv1alpha1 "openebs.io/metac/apis/metacontroller/v1alpha1"
	versioned "openebs.io/metac/client/generated/clientset/versioned"
	internalinterfaces "openebs.io/metac/client/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "openebs.io/metac/client/generated/listers/metacontroller/v1alpha1"
)

// GenericControllerInformer provides access to a shared informer and lister for
// GenericControllers.
type GenericControllerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.GenericControllerLister
}

type genericControllerInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewGenericControllerInformer constructs a new informer for GenericController type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewGenericControllerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredGenericControllerInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredGenericControllerInformer constructs a new informer for GenericController type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredGenericControllerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MetacontrollerV1alpha1().GenericControllers(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MetacontrollerV1alpha1().GenericControllers(namespace).Watch(options)
			},
		},
		&metacontrollerv1alpha1.GenericController{},
		resyncPeriod,
		indexers,
	)
}

func (f *genericControllerInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredGenericControllerInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *genericControllerInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&metacontrollerv1alpha1.GenericController{}, f.defaultInformer)
}

func (f *genericControllerInformer) Lister() v1alpha1.GenericControllerLister {
	return v1alpha1.NewGenericControllerLister(f.Informer().GetIndexer())
}
