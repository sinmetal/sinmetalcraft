/// <reference path="../typings/angularjs/angular.d.ts" />
/// <reference path="../typings/angularjs/angular-route.d.ts" />
/// <reference path="../typings/angularjs/angular-sanitize.d.ts" />

/// <reference path="../typings/angular-translate/angular-translate.d.ts" />
/// <reference path="../typings/angular-material/angular-material.d.ts" />
/// <reference path="../typings/angular-storage/angular-storage.d.ts" />

/// <reference path="./model.ts" />

/// <reference path="./controller/base_controller.ts" />
/// <reference path="./controller/example_list_controller.ts" />
/// <reference path="./controller/example_detail_controller.ts" />
/// <reference path="./controller/example_entry_controller.ts" />
/// <reference path="./controller/example_edit_controller.ts" />

/// <reference path="./service/example_service.ts" />
/// <reference path="./service/login_service.ts" />
/// <reference path="./service/server_service.ts" />

namespace SinmetalCraft {
    "use strict";

    export var appName = "sinmetalcraft";

    export interface IRootScope extends angular.IRootScopeService {
    }

    var rootDependencies:string[] = [
        "ngAnimate",
        "ngSanitize",
        "ngMaterial",
        "md.data.table",
        "pascalprecht.translate",
        appName + ".route",
        appName + ".run",
        appName + ".service",
        appName + ".controller",
        appName + ".directive",
    ];

    angular.module(
        appName,
        rootDependencies
    );

    angular.module(
        appName + ".route",
        ["ngRoute", "pascalprecht.translate"],
        ($httpProvider:angular.IHttpProvider, $routeProvider:angular.route.IRouteProvider, $locationProvider:angular.ILocationProvider, $translateProvider:angular.translate.ITranslateProvider) => {
            $httpProvider.interceptors.push(($q:angular.IQService, $injector:angular.auto.IInjectorService) => {
                var retryMaxCount = 2;

                return {
                    "responseError": (response:any) => {
                        var status = response.status;
                        if (status / 100 === 5) {
                            if (!response.config.retryCount) {
                                response.config.retryCount = 0;
                            }
                            if (response.config.retryCount < retryMaxCount) {
                                response.config.retryCount++;
                                var $http:any = $injector.get("$http");
                                return $http(response.config);
                            }
                        }

                        return $q.reject(response);
                    }
                };
            });

            $routeProvider
                // Example
                .when("/app/example", {
                    template: "<examplelist></examplelist>"
                })
                .when("/app/example/entry", {
                    template: "<exampleentry></exampleentry>"
                })
                .when("/app/example/edit/:key", {
                    template: "<exampleedit></exampleedit>"
                })
                .when("/app/example/:key", {
                    template: "<exampledetail></exampledetail>"
                })
                // その他
                .otherwise({
                    template: "<examplelist></examplelist>"
                });

            // hashの書き換えの代わりにHTML5のHistory API関係を使うモードを設定する。
            $locationProvider.html5Mode({
                enabled: true,
                requireBase: false
            });
        }
    );

    angular.module(
        appName + ".run",
        []
    );

    angular.module(
        appName + ".service",
        ["angular-storage"]
    )
        .service("loginService", SinmetalCraft.LoginService)
        .service("exampleService", SinmetalCraft.ExampleService)
        .service("serverService", SinmetalCraft.ServerService);


    angular.module(
        appName + ".controller",
        []
    )
        .controller("ExampleListController", SinmetalCraft.Controller.Example.List.Controller)
        .controller("ExampleDetailController", SinmetalCraft.Controller.Example.Detail.Controller)
        .controller("ExampleEntryController", SinmetalCraft.Controller.Example.Entry.Controller);

    angular.module(
        appName + ".directive",
        []
    )
        .directive("examplelist", SinmetalCraft.Controller.Example.List.directive)
        .directive("exampledetail", SinmetalCraft.Controller.Example.Detail.directive)
        .directive("exampleentry", SinmetalCraft.Controller.Example.Entry.directive)
        .directive("exampleedit", SinmetalCraft.Controller.Example.Edit.directive);
}
