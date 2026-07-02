/*
Copyright 2025.

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

package serviceprovider

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiv1alpha1 "github.com/openmcp-project/opencontrolplane-runtime/testdata/api/v1alpha1"
	configv1alpha1 "github.com/openmcp-project/opencontrolplane-runtime/testdata/config/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Foo Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		ctx := context.Background()
		providerConfigKey := types.NamespacedName{Name: "foo"}
		providerConfig := &configv1alpha1.ProviderConfig{}

		BeforeEach(func() {
			By("create a provider config if not found")
			err := platformClient.Get(ctx, providerConfigKey, providerConfig)
			if err != nil && errors.IsNotFound(err) {
				config := &configv1alpha1.ProviderConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
				}
				Expect(platformClient.Create(ctx, config)).To(Succeed())
			}
			// reset mock flag
			reconciler.createOrUpdateCalled = false
		})

		It("should successfully reconcile Foo", func() {
			By("Reconciling the created resource")
			foo := &apiv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
			}
			Expect(onboardingClient.Create(ctx, foo)).To(Succeed())
			Eventually(func() bool { return reconciler.createOrUpdateCalled }).Should(BeTrue())
			// verify poll interval default is applied on initial create
			Eventually(func() time.Duration { return reconciler.config.PollInterval() }).Should(Equal(time.Minute))
		})

		It("should receive a reconcile request when the provider config changes", func() {
			By("Reconciling the existing resource")
			config := &configv1alpha1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			}
			Expect(platformClient.Get(ctx, providerConfigKey, config)).To(Succeed())
			config.Spec.PollInterval = &metav1.Duration{Duration: time.Hour}
			Expect(platformClient.Update(ctx, config)).To(Succeed())
			Eventually(func() bool { return reconciler.createOrUpdateCalled }).Should(BeTrue())
			// verify a reconcile request has been created by matching the updated poll interval
			Eventually(func() time.Duration { return reconciler.config.PollInterval() }).Should(Equal(time.Hour))
		})
	})
})
