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
var _ = Describe("Buyer Controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		Namespace  = "buyer-controller-test"
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

	Context("When create buyer", func() {
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

		It("Should update buyer and corresponding seller", func() {
			By("Creating a new Buyer")
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
					Amount: &SpecAmount,
					Price:  SpecPrice,
				},
			}

			Expect(k8sClient.Create(ctx, buyer)).Should(Succeed())

			By("Found the created buyer")
			buyerLookupKey := types.NamespacedName{Name: BuyerName, Namespace: Namespace}
			createdBuyer := &v1alpha1.Buyer{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, buyerLookupKey, createdBuyer)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdBuyer.Spec.Name).Should(Equal(SpecName))
			Expect(*createdBuyer.Spec.Amount).Should(Equal(int32(10)))

			By("Creating a new seller with same product name")
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
					Amount: &OneAmount,
					Price:  SpecPrice,
					Money:  "0.00",
				},
			}

			Expect(k8sClient.Create(ctx, seller)).Should(Succeed())

			By("Reconciling the custom resources created")
			buyerReconciler := &BuyerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := buyerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: buyerLookupKey,
			})

			Expect(err).To(Not(HaveOccurred()))

			By("By checking that the buyer amount(demanded) decrease")

			Eventually(func() bool {
				err := k8sClient.Get(ctx, buyerLookupKey, createdBuyer)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdBuyer.Spec.Name).Should(Equal(SpecName))
			Expect(*createdBuyer.Spec.Amount).Should(Equal(int32(9)))

			By("By checking that the seller amount(available) decrease and money increase")
			sellerLookupKey := types.NamespacedName{Name: SellerName, Namespace: Namespace}
			createdSeller := &v1alpha1.Seller{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, sellerLookupKey, createdSeller)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdSeller.Spec.Name).Should(Equal(SpecName))
			Expect(*createdSeller.Spec.Amount).Should(Equal(int32(0)))
			Expect(createdSeller.Spec.Money).Should(Equal("100"))
		})
	})
})
