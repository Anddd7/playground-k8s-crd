/*
Copyright The Kubernetes Authors.

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
	"context"
	time "time"

	playgroundk8scrdv1alpha1 "github.com/Anddd7/playground-k8s-crd/pkg/apis/playgroundk8scrd/v1alpha1"
	versioned "github.com/Anddd7/playground-k8s-crd/pkg/generated/clientset/versioned"
	internalinterfaces "github.com/Anddd7/playground-k8s-crd/pkg/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/Anddd7/playground-k8s-crd/pkg/generated/listers/playgroundk8scrd/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// SellerInformer provides access to a shared informer and lister for
// Sellers.
type SellerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.SellerLister
}

type sellerInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewSellerInformer constructs a new informer for Seller type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewSellerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredSellerInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredSellerInformer constructs a new informer for Seller type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredSellerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Playgroundk8scrdV1alpha1().Sellers(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Playgroundk8scrdV1alpha1().Sellers(namespace).Watch(context.TODO(), options)
			},
		},
		&playgroundk8scrdv1alpha1.Seller{},
		resyncPeriod,
		indexers,
	)
}

func (f *sellerInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredSellerInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *sellerInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&playgroundk8scrdv1alpha1.Seller{}, f.defaultInformer)
}

func (f *sellerInformer) Lister() v1alpha1.SellerLister {
	return v1alpha1.NewSellerLister(f.Informer().GetIndexer())
}
