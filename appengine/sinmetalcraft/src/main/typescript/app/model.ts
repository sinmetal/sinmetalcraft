namespace SinmetalCraft.Model {
    "use strict";

    export interface IPair<T> {
        key: string;
        value: T;
    }

    /**
     * サーバの返却値がリストの場合このinterfaceで包む
     * @namespace
     * @property {string} ICursorList.cursor カーソル
     * @property {boolean} ICursorList.hasNext 続きがあるかどうか。trueの場合カーソルを使って取得できる
     * @property {T[]} ICursorList.list
     */
    export interface ICursorList<T> {
        cursor: string;
        hasNext: boolean;
        items: T[];
    }

    /**
     * cursorとlimit
     */
    export interface IListOptions {
        limit?: number;
        cursor?: string;
    }

    /**
     * Local Storageに保存する値
     */
    export interface IStorageBody {
        expire: number;
        body: any;
    }

    export interface IExample {
        key?: string;
        name: string;
        world: string;
        resourceID: number;
        zone: string;
        ipAddr: string;
        status: string;
        operationType: string;
        operationStatus: string;
        createdAt?: string;
        updatedAt?: string;
    }

    export interface IMessage {
        message: string;
    }

    export interface IListExampleRequest {
        cursor: string;
        limit: number;
    }

    export interface IExampleInsertRequest {
        name: string;
    }

    export interface IExampleUpdateRequest {
        key: string;
        name: string;
    }

    export interface IExampleDeleteRequest {
        key: string;
    }

    export interface ILogin {
        loginURL: string;
    }

    export interface IServerInsertRequest {
        key: string;
    }

    export interface IServerUpdateRequest {
        key: string;
        operation: string;
    }
}
