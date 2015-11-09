namespace SinmetalCraft.Controller.Base {
    "use strict";

    export class Controller {

        constructor(public $rootScope:SinmetalCraft.IRootScope, public $mdToast:angular.material.IToastService, public loginService:SinmetalCraft.LoginService, public $filter?:angular.IFilterService) {
        }

        showErrorToast(error:any):void {
            console.error(error);

            this.$mdToast.show(
                this.$mdToast.simple()
                    .content("Error, Please Reload")
                    .position("0 0")
                    .hideDelay(10000)
            );
        }

        showAuthToast(destinationURL:string):void {
            var toast = this.$mdToast.simple()
                .content("Authentication Error")
                .action("OK")
                .highlightAction(false)
                .hideDelay(30000)
                .position("0 0");
            this.$mdToast.show(toast).then((response) => {
                if (response === "ok") {
                    this.loginService.get(destinationURL)
                        .success((data)=> {
                            window.location.href = data.loginURL;
                        })
                        .error((_, status) => {
                            if (status === 0) {
                                // ページリロードなどによる中断
                                return;
                            }
                            this.showErrorToast(status);
                        });
                }
            });
        }
    }
}
