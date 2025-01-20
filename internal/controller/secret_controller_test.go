/*
Copyright 2025 drumato.

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

package controller

import (
	"log"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Secret Controller", func() {
	Context("Reconcile", func() {
		Context("When ttl was speficied", func() {
			It("should successfully delete the secret", func() {
				const secretName = "secretcontroller-ttl-succeed"
				req := ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      secretName,
					},
				}
				now := time.Now().UTC()
				secret := corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      secretName,
						Annotations: map[string]string{
							TTLAnnotation:       now.Add(2 * time.Second).Format(time.DateTime),
							TTLLayoutAnnotation: "datetime",
						},
					},
				}
				Eventually(func() error {
					return k8sClient.Create(ctx, &secret)
				}).WithTimeout(30 * time.Second).Should(Succeed())

				reconciler := &SecretReconciler{Client: k8sClient}

				By("checking the secret resource is actually deleted")
				Eventually(func() bool {
					_, err := reconciler.Reconcile(ctx, req)
					if err != nil {
						log.Println(err)
						return false
					}
					err = k8sClient.Get(ctx, req.NamespacedName, &secret)
					return apierrors.IsNotFound(err)
				}).WithTimeout(30 * time.Second).Should(BeTrue())
			})
		})

		Context("When ttl-after-age was speficied", func() {
			It("should successfully delete the secret", func() {
				const secretName = "secretcontroller-ttl-succeed"
				req := ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      secretName,
					},
				}
				secret := corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      secretName,
						Annotations: map[string]string{
							TTLAfterAgeAnnotation: "1s",
						},
					},
				}
				Eventually(func() error {
					return k8sClient.Create(ctx, &secret)
				}).WithTimeout(30 * time.Second).Should(Succeed())

				reconciler := &SecretReconciler{Client: k8sClient}

				By("checking the secret resource is actually deleted")
				Eventually(func() bool {
					_, err := reconciler.Reconcile(ctx, req)
					if err != nil {
						log.Println(err)
						return false
					}
					err = k8sClient.Get(ctx, req.NamespacedName, &secret)
					return apierrors.IsNotFound(err)
				}).WithTimeout(30 * time.Second).Should(BeTrue())
			})
		})
	})

	Context("parseDuration", func() {
		Context("3d", func() {
			It("shoud succeed", func() {
				d, err := parseDuration("3d")
				Expect(err).Should(Succeed())

				Expect(d).Should(BeIdenticalTo(3 * 24 * time.Hour))

			})
		})

		Context("4h", func() {
			It("shoud succeed", func() {
				d, err := parseDuration("4h")
				Expect(err).Should(Succeed())

				Expect(d).Should(BeIdenticalTo(4 * time.Hour))
			})
		})

		Context("5m", func() {
			It("shoud succeed", func() {
				d, err := parseDuration("5m")
				Expect(err).Should(Succeed())

				Expect(d).Should(BeIdenticalTo(5 * time.Minute))
			})
		})

		Context("6s", func() {
			It("shoud succeed", func() {
				d, err := parseDuration("6s")
				Expect(err).Should(Succeed())

				Expect(d).Should(BeIdenticalTo(6 * time.Second))
			})
		})
	})
})
