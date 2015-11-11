namespace SinmetalCraft.Controller.Example.Detail {
    "use strict";

    import Base = SinmetalCraft.Controller.Base;

    export function directive() {
        return {
            restrict: "E",
            controller: Controller,
            controllerAs: "exa",
            scope: {},
            templateUrl: "/template/example_detail.html"
        };
    }

    export class Controller extends Base.Controller {

        store:IExampleStore;

        constructor(public $scope:angular.IScope, public $rootScope:SinmetalCraft.IRootScope, public $routeParams:angular.route.IRouteParamsService, public $location:angular.ILocationService, public $mdDialog:ng.material.IDialogService, public $mdToast:angular.material.IToastService, public exampleService:SinmetalCraft.ExampleService, public loginService:SinmetalCraft.LoginService) {
            super($rootScope, $mdToast, loginService);

            this.store = this.exampleService.exampleStore;

            var key:string = $routeParams["key"];
            if (!this.store.example) {
                this.exampleService.get(key)
                    .success(data=> {
                        this.store.example = data;
                    })
                    .error((_, status) => {
                        if (status === 0) {
                            // ページリロードなどによる中断
                            return;
                        }
                        if (status === 401) {
                            this.showAuthToast("/app/example/" + this.store.example.key);
                            return;
                        }
                        this.showErrorToast(status);
                    });
            }

            Controller.$inject = ["$scope", "$rootScope", "$routeParams", "$location", "$mdDialog", "$mdToast", "exampleService", "loginService"];
        }

        showConfirm(ev:MouseEvent) {
            var confirm = this.$mdDialog.confirm()
                .title("Delete Example")
                .content("Are you sure you want to delete ?")
                .ariaLabel("Delete Example")
                .targetEvent(ev)
                .ok("Remove example")
                .cancel("Cancel");
            this.$mdDialog.show(confirm).then(() => {
                this.exampleService.delete({
                    key: this.store.example.key
                })
                    .success(() => {
                        this.store.examples.items = this.store.examples.items.filter(v=> {
                            return (v.key !== this.store.example.key);
                        });
                        this.$location.path("/app/example/");
                    })
                    .error((err, status) => {
                        if (status === 0) {
                            // ページリロードなどによる中断
                            return;
                        }
                        if (status === 401) {
                            this.showAuthToast("/app/example/" + this.store.example.key);
                            return;
                        }
                        this.showErrorToast(status);
                    });
            }, () => {
                // cancel, no-op
            });
        }

        viewEditPage():void {
            this.$location.path("/app/example/edit/" + this.store.example.key);
        }

        backToList():void {
            this.$location.path("/app/example/");
        }
    }
}

