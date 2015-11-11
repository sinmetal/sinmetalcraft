namespace SinmetalCraft.Controller.Example.List {
    "use strict";

    import Base = SinmetalCraft.Controller.Base;

    export function directive() {
        return {
            restrict: "E",
            controller: Controller,
            controllerAs: "exa",
            scope: {},
            templateUrl: "/template/example_list.html"
        };
    }

    export class Controller extends Base.Controller {

        store:IExampleStore;

        constructor(public $scope:angular.IScope, public $rootScope:SinmetalCraft.IRootScope, public $routeParams:angular.route.IRouteParamsService, public $location:angular.ILocationService, public $mdToast:angular.material.IToastService, public exampleService:SinmetalCraft.ExampleService, public loginService:SinmetalCraft.LoginService, public serverService:SinmetalCraft.ServerService) {
            super($rootScope, $mdToast, loginService);

            this.store = exampleService.exampleStore;

            if (this.store.examples.items.length < 1) {
                this.list(20);
            }

            Controller.$inject = ["$scope", "$rootScope", "$routeParams", "$location", "$mdToast", "exampleService", "loginService", "serverService"];
        }

        list(limit:number = null):void {
            var request:SinmetalCraft.Model.IListExampleRequest = {
                cursor: null,
                limit: limit
            };

            this.exampleService.list(request)
                .success(data=> {
                    this.store.examples.items = data;
                })
                .error((_, status) => {
                    if (status === 0) {
                        // ページリロードなどによる中断
                        return;
                    }
                    if (status === 401) {
                        this.showAuthToast("/app/example/");
                        return;
                    }
                    this.showErrorToast(status);
                });
        }

        readMoreList(limit:number = null):void {
            var request:SinmetalCraft.Model.IListExampleRequest = {
                cursor: this.store.examples.cursor,
                limit: limit
            };

            this.exampleService.list(request)
                .success(data=> {
                })
                .error((_, status) => {
                    if (status === 0) {
                        // ページリロードなどによる中断
                        return;
                    }
                    if (status === 401) {
                        this.showAuthToast("/app/example/");
                        return;
                    }
                    this.showErrorToast(status);
                });
        }

        startServer(index:number):void {
            this.store.example = this.store.examples.items[index];
            var request : SinmetalCraft.Model.IServerInsertRequest = {
                "key" : this.store.example.key
            };

            this.serverService.insert(request)
                .success(data=> {
                    this.showServerStartToast();
                })
                .error((_, status) => {
                    if (status === 0) {
                        // ページリロードなどによる中断
                        return;
                    }
                    if (status === 401) {
                        this.showAuthToast("/");
                        return;
                    }
                    this.showErrorToast(status);
                });
        }

        viewEntryPage():void {
            this.$location.path("/app/example/entry");
        }

        backToList():void {
            this.$location.path("/app/example");
        }

        showServerStartToast():void {
            this.$mdToast.show(
                this.$mdToast.simple()
                    .content("Server Start")
                    .position("0 0")
                    .hideDelay(10000)
            );
        }
    }
}

