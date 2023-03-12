package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	crdv1alpha1 "github.com/Anddd7/playground-k8s-crd/pkg/apis/playgroundk8scrd/v1alpha1"
	clientset "github.com/Anddd7/playground-k8s-crd/pkg/generated/clientset/versioned"
	crdscheme "github.com/Anddd7/playground-k8s-crd/pkg/generated/clientset/versioned/scheme"
	informers "github.com/Anddd7/playground-k8s-crd/pkg/generated/informers/externalversions/playgroundk8scrd/v1alpha1"
	listers "github.com/Anddd7/playground-k8s-crd/pkg/generated/listers/playgroundk8scrd/v1alpha1"
)

const controllerAgentName = "github.com/Anddd7/playground-k8s-crd-controller"

const (
	SuccessSynced      = "Synced"
	WarnNoSellerExists = "WarnNoSellerExists"
	WarnNoBuyerExists  = "WarnNoBuyerExists"
)

// Controller is the controller implementation for Foo resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	crdclientset clientset.Interface

	buyersLister listers.BuyerLister
	buyersSynced cache.InformerSynced

	sellersLister listers.SellerLister
	sellersSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	crdclientset clientset.Interface,
	buyerInformer informers.BuyerInformer,
	sellerInformer informers.SellerInformer,
) *Controller {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(crdscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset: kubeclientset,
		crdclientset:  crdclientset,
		buyersLister:  buyerInformer.Lister(),
		buyersSynced:  buyerInformer.Informer().HasSynced,
		sellersLister: sellerInformer.Lister(),
		sellersSynced: sellerInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "BuyerAndSeller"),
		recorder:      recorder,
	}

	klog.Info("Setting up event handlers")

	// Set up an event handler for when Foo resources change
	buyerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueBuyer,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueBuyer(new)
		},
	})

	sellerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueSeller,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueSeller(new)
		},
	})

	// deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
	// 	AddFunc: controller.handleObject,
	// 	UpdateFunc: func(old, new interface{}) {
	// 		newDepl := new.(*appsv1.Deployment)
	// 		oldDepl := old.(*appsv1.Deployment)
	// 		if newDepl.ResourceVersion == oldDepl.ResourceVersion {
	// 			// Periodic resync will send update events for all known Deployments.
	// 			// Two different versions of the same Deployment will always have different RVs.
	// 			return
	// 		}
	// 		controller.handleObject(new)
	// 	},
	// 	DeleteFunc: controller.handleObject,
	// })

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting github.com/Anddd7/playground-k8s-crd controller")
	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.buyersSynced, c.sellersSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	// CRD: get crd type from the key
	parts := strings.Split(key, "/")
	if len(parts) != 3 {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	crdType, namespace, name := parts[0], parts[1], parts[2]
	if crdType == "Buyer" {
		buyer, err := c.buyersLister.Buyers(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				utilruntime.HandleError(fmt.Errorf("'%s' in work queue no longer exists", key))
				return nil
			}
			return err
		}

		sellers, err := c.findSeller(namespace, buyer.Spec.Name, buyer.Spec.Price)
		if err != nil {
			c.recorder.Event(
				buyer, corev1.EventTypeWarning, WarnNoSellerExists,
				err.Error(),
			)
		} else {
			c.buy(buyer, sellers)
			c.recorder.Event(buyer, corev1.EventTypeNormal, SuccessSynced, "Buyer synced successfully")
		}
	}
	if crdType == "Seller" {
		seller, err := c.sellersLister.Sellers(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				utilruntime.HandleError(fmt.Errorf("'%s' in work queue no longer exists", key))
				return nil
			}
			return err
		}

		buyers, err := c.findBuyer(namespace, seller.Spec.Name, seller.Spec.Price)
		if err != nil {
			c.recorder.Event(
				seller, corev1.EventTypeWarning, WarnNoBuyerExists,
				err.Error(),
			)
		} else {
			c.sell(seller, buyers)
			c.recorder.Event(seller, corev1.EventTypeNormal, SuccessSynced, "Seller synced successfully")
		}
	}

	return nil
}

func (c *Controller) findBuyer(namespace, productName string, price *float32) ([]*crdv1alpha1.Buyer, error) {
	list, err := c.buyersLister.Buyers(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	ans := []*crdv1alpha1.Buyer{}
	for _, item := range list {
		if item.Spec.Name == productName && *item.Spec.Price >= *price {
			ans = append(ans, item)
		}
	}
	if len(ans) == 0 {
		return ans, fmt.Errorf("no suitable buyer of %s exists", productName)
	}
	return ans, nil
}
func (c *Controller) findSeller(namespace, productName string, price *float32) ([]*crdv1alpha1.Seller, error) {
	list, err := c.sellersLister.Sellers(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	ans := []*crdv1alpha1.Seller{}
	for _, item := range list {
		if item.Spec.Name == productName && *item.Spec.Price <= *price && *item.Spec.Amount > 0 {
			ans = append(ans, item)
		}
	}
	if len(ans) == 0 {
		return ans, fmt.Errorf("no suitable seller of %s exists", productName)
	}
	return ans, nil
}

func (c *Controller) buy(buyer *crdv1alpha1.Buyer, sellers []*crdv1alpha1.Seller) error {
	amount, price := *buyer.Spec.Amount, *buyer.Spec.Price

	for _, seller := range sellers {
		if amount == 0 {
			break
		}
		settleAmount := int32(math.Min(float64(*seller.Spec.Amount), float64(amount)))
		restAmount := *seller.Spec.Amount - settleAmount
		money := *seller.Spec.Money + float32(settleAmount)*price

		sellerCopy := seller.DeepCopy()
		sellerCopy.Spec.Amount = &restAmount
		sellerCopy.Spec.Money = &money

		_, err := c.crdclientset.Playgroundk8scrdV1alpha1().Sellers(seller.Namespace).Update(context.TODO(), sellerCopy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		klog.Info(
			fmt.Sprintf("%s: buy %s from %s, with %f * %d",
				buyer.Name, buyer.Spec.Name, seller.Name, price, settleAmount,
			))

		amount -= settleAmount
	}

	buyerCopy := buyer.DeepCopy()
	buyerCopy.Spec.Amount = &amount

	_, err := c.crdclientset.Playgroundk8scrdV1alpha1().Buyers(buyer.Namespace).Update(context.TODO(), buyerCopy, metav1.UpdateOptions{})
	return err
}

func (c *Controller) sell(seller *crdv1alpha1.Seller, buyers []*crdv1alpha1.Buyer) error {
	amount, money := *seller.Spec.Amount, *seller.Spec.Money

	for _, buyer := range buyers {
		if amount == 0 {
			break
		}
		settleAmount := int32(math.Min(float64(*buyer.Spec.Amount), float64(amount)))
		restAmount := *buyer.Spec.Amount - settleAmount
		price := *buyer.Spec.Price

		buyerCopy := buyer.DeepCopy()
		buyerCopy.Spec.Amount = &restAmount

		_, err := c.crdclientset.Playgroundk8scrdV1alpha1().Buyers(buyer.Namespace).Update(context.TODO(), buyerCopy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		klog.Info(
			fmt.Sprintf("%s: sell %s to %s, with %f * %d",
				seller.Name, seller.Spec.Name, buyer.Name, price, settleAmount,
			))

		amount -= settleAmount
		money += float32(settleAmount) * price
	}

	sellerCopy := seller.DeepCopy()
	sellerCopy.Spec.Amount = &amount
	sellerCopy.Spec.Money = &money

	_, err := c.crdclientset.Playgroundk8scrdV1alpha1().Sellers(seller.Namespace).Update(context.TODO(), sellerCopy, metav1.UpdateOptions{})
	return err
}

// enqueueFoo takes a Foo resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Foo.
func (c *Controller) enqueueBuyer(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add("Buyer/" + key)
}

func (c *Controller) enqueueSeller(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add("Seller/" + key)
}
