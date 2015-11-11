namespace SinmetalCraft.Controller.Example.Entry {
    "use strict";

    import Base = SinmetalCraft.Controller.Base;
    import IExampleInsertRequest = SinmetalCraft.Model.IExampleInsertRequest;

    export function directive() {
        return {
            restrict: "E",
            controller: Controller,
            controllerAs: "exa",
            scope: {},
            templateUrl: "/template/example_entry.html"
        };
    }

    export class Controller extends Base.Controller {

        param:IExampleInsertRequest;
        exampleStore:IExampleStore;

        constructor(public $scope:angular.IScope, public $rootScope:SinmetalCraft.IRootScope, public $routeParams:angular.route.IRouteParamsService, public $location:angular.ILocationService, public $mdToast:angular.material.IToastService, public exampleService:SinmetalCraft.ExampleService, public loginService:SinmetalCraft.LoginService) {
            super($rootScope, $mdToast, loginService);

            this.exampleStore = this.exampleService.exampleStore;

            this.param = this.exampleService.getDraftForInsert();

            Controller.$inject = ["$scope", "$rootScope", "$routeParams", "$location", "$mdToast", "exampleService", "loginService"];
        }

        entry():void {
            this.exampleService.insert(this.param)
                .success(data=> {
                    this.exampleStore.examples.items.unshift(data);
                    this.exampleService.removeDraftForInsert();
                    this.$location.path("/app/example/");
                })
                .error((_, status) => {
                    if (status === 0) {
                        // ページリロードなどによる中断
                        return;
                    }
                    if (status === 401) {
                        this.exampleService.setDraftForInsert(this.param);
                        this.showAuthToast("/app/example/entry");
                        return;
                    }
                    this.showErrorToast(status);
                });
        }

        backToList():void {
            this.$location.path("/app/example/");
        }
    }
}

