/*
Copyright 2023.

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

package controllers

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"math"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"

	transactionv1alpha1 "github.com/Anddd7/playground-k8s-crd/api/v1alpha1"
)

// BuyerReconciler reconciles a Buyer object
type BuyerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=transaction.mesh-shaped.top,resources=buyers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=transaction.mesh-shaped.top,resources=buyers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=transaction.mesh-shaped.top,resources=buyers/finalizers,verbs=update
//+kubebuilder:rbac:groups=transaction.mesh-shaped.top,resources=sellers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=transaction.mesh-shaped.top,resources=sellers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=transaction.mesh-shaped.top,resources=sellers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Buyer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *BuyerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	buyer := &transactionv1alpha1.Buyer{}
	err := r.Get(ctx, req.NamespacedName, buyer)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	sellers, err := r.findSellers(ctx, buyer)
	if err != nil {
		log.Info(err.Error())
		return ctrl.Result{}, nil
	}
	log.Info(
		fmt.Sprintf("Found %d sellers, start to buy %s from them", len(sellers), buyer.Spec.Name),
	)

	amount := *buyer.Spec.Amount
	price, _ := strconv.ParseFloat(buyer.Spec.Price, 32)
	for _, seller := range sellers {
		if amount == 0 {
			break
		}
		settleAmount := int32(math.Min(float64(*seller.Spec.Amount), float64(amount)))

		// buyer
		restAmount := *seller.Spec.Amount - settleAmount
		amount -= settleAmount

		// seller
		money, _ := strconv.ParseFloat(seller.Spec.Money, 32)
		money += float64(settleAmount) * price

		// update
		seller.Spec.Amount = &restAmount
		seller.Spec.Money = strconv.FormatFloat(money, 'f', -1, 32)
		_ = r.Update(ctx, seller)
	}

	// update
	buyer.Spec.Amount = &amount
	_ = r.Update(ctx, buyer)

	log.Info(
		fmt.Sprintf("Finished buy transaction, rest amount is %d", amount),
	)

	return ctrl.Result{}, nil
}

func (r *BuyerReconciler) findSellers(ctx context.Context, buyer *transactionv1alpha1.Buyer) ([]*transactionv1alpha1.Seller, error) {
	log := log.FromContext(ctx)
	var sellers transactionv1alpha1.SellerList
	listOpts := []client.ListOption{
		client.InNamespace(buyer.Namespace),
		//client.MatchingFields{"spec.name": buyer.Spec.Name},
	}
	_ = r.List(ctx, &sellers, listOpts...)

	log.Info(
		fmt.Sprintf("Find %d sellers who is selling %s", len(sellers.Items), buyer.Spec.Name),
	)
	var ans []*transactionv1alpha1.Seller
	for _, seller := range sellers.Items {
		if *seller.Spec.Amount > 0 &&
			seller.Spec.Name == buyer.Spec.Name &&
			seller.Spec.Price <= buyer.Spec.Price {
			ans = append(ans, &seller)
		}
	}
	if len(ans) == 0 {
		return ans, fmt.Errorf("no suitable seller of %s exists", buyer.Spec.Name)
	}
	return ans, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuyerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&transactionv1alpha1.Buyer{}).
		Complete(r)
}
