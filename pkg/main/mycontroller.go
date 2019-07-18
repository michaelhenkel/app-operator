package main

import (
	"flag"
	"fmt"
	"time"

	"k8s.io/klog"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	indexerMap  map[string]*cache.Indexer
	informerMap map[string]*cache.SharedIndexInformer
	queue       workqueue.RateLimitingInterface
}

func NewController(queue workqueue.RateLimitingInterface, indexerMap map[string]*cache.Indexer, informerMap map[string]*cache.SharedIndexInformer) *Controller {
	return &Controller{
		informerMap: informerMap,
		indexerMap:  indexerMap,
		queue:       queue,
	}
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.syncToStdout(key.(string))
	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(err, key)
	return true
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the pod to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) syncToStdout(key string) error {
	podIndexer := *c.indexerMap["pod"]
	deploymentIndexer := *c.indexerMap["deployment"]
	obj, exists, err := podIndexer.GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		obj, exists, err = deploymentIndexer.GetByKey(key)
		if err != nil {
			klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
			return err
		}
	}
	if !exists {
		fmt.Printf("Object doesn't exist %s\n", key)
		return nil
	}
	switch obj.(type) {
	case *v1.Pod:
		fmt.Printf("Sync/Add/Update for Pod %s\n", obj.(*v1.Pod).GetName())
	case *appsv1.Deployment:
		fmt.Printf("Sync/Add/Update for Deployment %s\n", obj.(*appsv1.Deployment).GetName())
	default:
		fmt.Printf("no idea %+v\n", obj)
	}

	return nil
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		klog.Infof("Error syncing pod %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	klog.Infof("Dropping pod %q out of the queue: %v", key, err)
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()

	for informerName, informer := range c.informerMap {
		klog.Info("Starting Informer ", informerName)
		inf := *informer
		go inf.Run(stopCh)
		//go c.informer.Run(stopCh)

		// Wait for all involved caches to be synced, before processing items from the queue is started
		if !cache.WaitForCacheSync(stopCh, inf.HasSynced) {
			runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
			return
		}
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("Stopping Pod controller")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func main() {
	var kubeconfig string
	var master string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
	flag.Parse()

	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	// create the pod watcher
	podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", v1.NamespaceDefault, fields.Everything())

	// create the deployment watcher
	deploymentListWatcher := cache.NewListWatchFromClient(clientset.AppsV1().RESTClient(), "deployments", v1.NamespaceDefault, fields.Everything())

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Pod than the version which was responsible for triggering the update.
	resourceEventHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}

	podInformer := cache.NewSharedIndexInformer(podListWatcher, &v1.Pod{}, 0, cache.Indexers{})
	podInformer.AddEventHandler(resourceEventHandlerFuncs)
	deploymentInformer := cache.NewSharedIndexInformer(deploymentListWatcher, &appsv1.Deployment{}, 0, cache.Indexers{})
	deploymentInformer.AddEventHandler(resourceEventHandlerFuncs)

	podIndexer := podInformer.GetIndexer()
	deploymentIndexer := deploymentInformer.GetIndexer()

	informerMap := map[string]*cache.SharedIndexInformer{"pod": &podInformer, "deployment": &deploymentInformer}
	indexerMap := map[string]*cache.Indexer{"pod": &podIndexer, "deployment": &deploymentIndexer}

	controller := NewController(queue, indexerMap, informerMap)

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	// Wait forever
	select {}
}
