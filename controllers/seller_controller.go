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
	"math"
	"strconv"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	transactionv1alpha1 "github.com/Anddd7/playground-k8s-crd/api/v1alpha1"
)

// SellerReconciler reconciles a Seller object
type SellerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=transaction.mesh-shaped.top,resources=sellers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=transaction.mesh-shaped.top,resources=sellers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=transaction.mesh-shaped.top,resources=sellers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Seller object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *SellerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	seller := &transactionv1alpha1.Seller{}
	err := r.Get(ctx, req.NamespacedName, seller)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	buyers, err := r.findBuyers(ctx, seller)
	if err != nil {
		log.Info(err.Error())
		return ctrl.Result{}, nil
	}
	log.Info(
		fmt.Sprintf("Found %d buyers, start to sell %s to them", len(buyers), seller.Spec.Name),
	)

	amount := *seller.Spec.Amount
	money, _ := strconv.ParseFloat(seller.Spec.Money, 32)
	for _, buyer := range buyers {
		if amount == 0 {
			break
		}
		settleAmount := int32(math.Min(float64(*buyer.Spec.Amount), float64(amount)))

		// buyer
		restAmount := *buyer.Spec.Amount - settleAmount
		price, _ := strconv.ParseFloat(buyer.Spec.Price, 32)

		// seller
		money += float64(settleAmount) * price
		amount -= settleAmount

		// update
		buyer.Spec.Amount = &restAmount
		_ = r.Update(ctx, buyer)
	}

	// update
	seller.Spec.Amount = &amount
	seller.Spec.Money = strconv.FormatFloat(money, 'f', -1, 32)
	_ = r.Update(ctx, seller)

	log.Info(
		fmt.Sprintf("Finished sell transaction, rest amount is %d, earned money is %f", amount, money),
	)

	return ctrl.Result{}, nil
}

func (r *SellerReconciler) findBuyers(ctx context.Context, seller *transactionv1alpha1.Seller) ([]*transactionv1alpha1.Buyer, error) {
	log := log.FromContext(ctx)
	var buyers transactionv1alpha1.BuyerList
	listOpts := []client.ListOption{
		client.InNamespace(seller.Namespace),
		//client.MatchingFields{"spec.name": seller.Spec.Name},
	}
	_ = r.List(ctx, &buyers, listOpts...)

	log.Info(
		fmt.Sprintf("Find %d buyers who want to buy %s", len(buyers.Items), seller.Spec.Name),
	)
	var ans []*transactionv1alpha1.Buyer
	for _, buyer := range buyers.Items {
		if *buyer.Spec.Amount > 0 &&
			seller.Spec.Name == buyer.Spec.Name &&
			seller.Spec.Price <= buyer.Spec.Price {
			ans = append(ans, &buyer)
		}
	}
	if len(ans) == 0 {
		return ans, fmt.Errorf("no suitable buyer of %s exists", seller.Spec.Name)
	}
	return ans, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SellerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&transactionv1alpha1.Seller{}).
		Complete(r)
}
