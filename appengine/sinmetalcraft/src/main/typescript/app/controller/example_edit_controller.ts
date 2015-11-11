namespace SinmetalCraft.Controller.Example.Edit {
    "use strict";

    import Base = SinmetalCraft.Controller.Base;
    import IExampleUpdateRequest = SinmetalCraft.Model.IExampleUpdateRequest;

    export function directive() {
        return {
            restrict: "E",
            controller: Controller,
            controllerAs: "exa",
            scope: {},
            templateUrl: "/template/example_edit.html"
        };
    }

    export class Controller extends Base.Controller {

        param:IExampleUpdateRequest;
        store:IExampleStore;

        constructor(public $scope:angular.IScope, public $rootScope:SinmetalCraft.IRootScope, public $routeParams:angular.route.IRouteParamsService, public $mdToast:angular.material.IToastService, public $location:angular.ILocationService, public exampleService:SinmetalCraft.ExampleService, public loginService:SinmetalCraft.LoginService) {
            super($rootScope, $mdToast, loginService);

            this.store = this.exampleService.exampleStore;

            var key:string = this.$routeParams["key"];
            this.exampleService.get(key)
                .success(data=> {
                    this.store.example = data;
                    this.param = {
                        "name": this.store.example.name,
                        "key": this.store.example.key
                    };
                    var draft = this.exampleService.getDraftForUpdate();
                    if (draft.key === key) {
                        this.param.name = draft.name;
                    }
                })
                .error((_, status) => {
                    if (status === 0) {
                        // ページリロードなどによる中断
                        return;
                    }
                    if (status === 401) {
                        this.showAuthToast("/app/example/edit/" + this.$routeParams["key"]);
                        return;
                    }
                    this.showErrorToast(status);
                });

            Controller.$inject = ["$scope", "$rootScope", "$routeParams", "$mdToast", "$location", "exampleService", "loginService"];
        }

        update():void {
            this.exampleService.update(this.param)
                .success(data=> {
                    this.exampleService.removeDraftForUpdate();
                    this.store.examples.items = this.store.examples.items.filter(function(v) {
                        return (v.key !== data.key);
                    });
                    this.store.examples.items.unshift(data);
                    this.$location.path("/app/example/");
                })
                .error((_, status) => {
                    if (status === 0) {
                        // ページリロードなどによる中断
                        return;
                    }
                    if (status === 401) {
                        this.exampleService.setDraftForUpdate(this.param);
                        this.showAuthToast("/app/example/edit/" + this.$routeParams["key"]);
                        return;
                    }
                    this.showErrorToast(status);
                });
        }

        backToDetail():void {
            this.$location.path("/app/example/" + this.$routeParams["key"]);
        }
    }
}
