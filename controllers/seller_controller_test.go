package controllers

import (
	"context"
	"time"

	"github.com/Anddd7/playground-k8s-crd/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// +kubebuilder:docs-gen:collapse=Imports
var _ = Describe("Seller Controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		Namespace  = "seller-controller-test"
		BuyerName  = "buyer-sample"
		SellerName = "seller-sample"
		SpecName   = "sample"
		SpecPrice  = "100.00"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	var SpecAmount int32 = 10
	var OneAmount int32 = 1

	Context("When create seller", func() {
		ctx := context.Background()
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      Namespace,
				Namespace: Namespace,
			},
		}

		BeforeEach(func() {
			err := k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace)
		})

		It("Should update seller and corresponding buyer", func() {
			By("Creating a new Seller")
			seller := &v1alpha1.Seller{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "transaction.mesh-shaped.top/v1alpha1",
					Kind:       "Seller",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      SellerName,
					Namespace: Namespace,
				},
				Spec: v1alpha1.SellerSpec{
					Name:   SpecName,
					Amount: &SpecAmount,
					Price:  SpecPrice,
					Money:  "0.00",
				},
			}

			Expect(k8sClient.Create(ctx, seller)).Should(Succeed())

			By("Found the created seller")
			sellerLookupKey := types.NamespacedName{Name: SellerName, Namespace: Namespace}
			createdSeller := &v1alpha1.Seller{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, sellerLookupKey, createdSeller)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdSeller.Spec.Name).Should(Equal(SpecName))
			Expect(*createdSeller.Spec.Amount).Should(Equal(int32(10)))

			By("Creating a new buyer with same product name")
			buyer := &v1alpha1.Buyer{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "transaction.mesh-shaped.top/v1alpha1",
					Kind:       "Buyer",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      BuyerName,
					Namespace: Namespace,
				},
				Spec: v1alpha1.BuyerSpec{
					Name:   SpecName,
					Amount: &OneAmount,
					Price:  SpecPrice,
				},
			}

			Expect(k8sClient.Create(ctx, buyer)).Should(Succeed())

			By("Reconciling the custom resources created")
			sellerReconciler := &SellerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := sellerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: sellerLookupKey,
			})

			Expect(err).To(Not(HaveOccurred()))

			By("By checking that the seller amount(available) decrease and money increase")

			Eventually(func() bool {
				err := k8sClient.Get(ctx, sellerLookupKey, createdSeller)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdSeller.Spec.Name).Should(Equal(SpecName))
			Expect(*createdSeller.Spec.Amount).Should(Equal(int32(9)))
			Expect(createdSeller.Spec.Money).Should(Equal("100"))

			By("By checking that the buyer amount(demanded) decrease")
			buyerLookupKey := types.NamespacedName{Name: BuyerName, Namespace: Namespace}
			createdBuyer := &v1alpha1.Buyer{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, buyerLookupKey, createdBuyer)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdBuyer.Spec.Name).Should(Equal(SpecName))
			Expect(*createdBuyer.Spec.Amount).Should(Equal(int32(0)))
		})
	})
})
