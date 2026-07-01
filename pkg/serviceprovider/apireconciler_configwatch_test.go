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
	"github.com/openmcp-project/opencontrolplane-runtime/test/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Foo Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name: "foo",
		}
		providerConfig := &v1alpha1.ProviderConfig{}

		BeforeEach(func() {
			By("create provider config instance foo")
			err := platformClient.Get(ctx, typeNamespacedName, providerConfig)
			if err != nil && errors.IsNotFound(err) {
				config := &v1alpha1.ProviderConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
				}
				Expect(platformClient.Create(ctx, config)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the config instance.
			config := &v1alpha1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			}
			err := platformClient.Get(ctx, typeNamespacedName, config)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup provider config instance foo")
			Expect(platformClient.Delete(ctx, config)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			foo := &v1alpha1.FooService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
			}
			Expect(onboardingClient.Create(ctx, foo)).To(Succeed())
			Eventually(reconciler.created, "10m").Should(Receive())
		})
		It("should receive a reconcile request when the provider config changes", func() {
			By("Reconciling the existing resource")
			config := &v1alpha1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			}
			Expect(platformClient.Get(ctx, typeNamespacedName, config)).To(Succeed())
			config.Spec.PollInterval = &metav1.Duration{Duration: time.Hour}
			Expect(platformClient.Update(ctx, config)).To(Succeed())
			Eventually(reconciler.created, "5s").Should(Receive(&v1alpha1.ProviderConfig{}, HaveField("Spec.PollInterval", &metav1.Duration{Duration: time.Hour})))
		})
	})
})
