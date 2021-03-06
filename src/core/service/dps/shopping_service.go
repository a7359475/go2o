/**
 * Copyright 2014 @ z3q.net.
 * name :
 * author : jarryliu
 * date : 2013-12-05 17:53
 * description :
 * history :
 */

package dps

import (
	"bytes"
	"errors"
	"go2o/src/core/domain/interface/enum"
	"go2o/src/core/domain/interface/shopping"
	"go2o/src/core/dto"
)

type shoppingService struct {
	_rep shopping.IShoppingRep
}

func NewShoppingService(r shopping.IShoppingRep) *shoppingService {
	return &shoppingService{
		_rep: r,
	}
}

func (this *shoppingService) BuildOrder(merchantId int, subject string, memberId int,
	cartKey string, couponCode string) (map[string]interface{}, error) {
	var sp shopping.IShopping = this._rep.GetShopping(merchantId)
	order, _, err := sp.BuildOrder(memberId, subject, couponCode)
	if err != nil {
		return nil, err
	}

	v := order.GetValue()
	buf := bytes.NewBufferString("")

	for _, v := range order.GetCoupons() {
		buf.WriteString(v.GetDescribe())
		buf.WriteString("\n")
	}

	var data map[string]interface{}
	data = make(map[string]interface{})
	if couponCode != "" {
		if v.CouponFee == 0 {
			data["result"] = v.CouponFee != 0
			data["message"] = "优惠券无效"
		} else {
			// 成功应用优惠券
			data["totalFee"] = v.TotalFee
			data["fee"] = v.Fee
			data["payFee"] = v.PayFee
			data["discountFee"] = v.DiscountFee
			data["couponFee"] = v.CouponFee
			data["couponDescribe"] = buf.String()
		}
	} else {
		//　取消优惠券
		data["totalFee"] = v.TotalFee
		data["fee"] = v.Fee
		data["payFee"] = v.PayFee
		data["discountFee"] = v.DiscountFee
	}
	return data, err
}

func (this *shoppingService) SubmitOrder(merchantId, memberId int, subject string, couponCode string, useBalanceDiscount bool) (
	orderNo string, err error) {
	var sp shopping.IShopping = this._rep.GetShopping(merchantId)
	return sp.SubmitOrder(memberId, subject, couponCode, useBalanceDiscount)
}

func (this *shoppingService) SetDeliverShop(merchantId int, orderNo string,
	shopId int) error {
	var sp shopping.IShopping = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err == nil {
		if err = order.SetShop(shopId); err == nil {
			_, err = order.Save()
		}
	}
	return err
}

func (this *shoppingService) HandleOrder(merchantId int, orderNo string) error {
	var sp shopping.IShopping = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err == nil {
		b := order.IsOver()
		if b {
			return errors.New("订单已经完成!")
		}

		status := order.GetValue().Status
		switch status + 1 {
		case enum.ORDER_WAIT_CONFIRM:
			err = order.Confirm()
		case enum.ORDER_WAIT_DELIVERY:
			err = order.Process()
		case enum.ORDER_WAIT_RECEIVE:
			err = order.Deliver(0, "")
		case enum.ORDER_RECEIVED:
			err = order.SignReceived()
		case enum.ORDER_COMPLETED:
			err = order.Complete()
		}
	}
	return err
}

// 根据编号获取订单
func (this *shoppingService) GetOrderById(id int) *shopping.ValueOrder {
	v := this._rep.GetOrderById(id)
	if v != nil {
		v.Items = this._rep.GetOrderItems(id)
	}
	return v
}

func (this *shoppingService) GetOrderByNo(merchantId int,
	orderNo string) *shopping.ValueOrder {
	var sp shopping.IShopping = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err != nil {
		return nil
	}
	if order == nil {
		return nil
	}
	v := order.GetValue()
	return &v
}

// 根据订单号获取订单
func (this *shoppingService) GetValueOrderByNo(orderNo string) *shopping.ValueOrder {
	return this._rep.GetValueOrderByNo(orderNo)
}

func (this *shoppingService) CancelOrder(merchantId int, orderNo string, reason string) error {
	var sp shopping.IShopping = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err == nil {
		err = order.Cancel(reason)
	}
	return err
}

//  获取购物车
func (this *shoppingService) getShoppingCart(merchantId int, buyerId int, cartKey string) shopping.ICart {
	sp := this._rep.GetShopping(merchantId)
	return sp.GetShoppingCart(buyerId, cartKey)
}

// 获取购物车,当购物车编号不存在时,将返回一个新的购物车
func (this *shoppingService) GetShoppingCart(merchantId int, memberId int, cartKey string) *dto.ShoppingCart {
	cart := this.getShoppingCart(merchantId, memberId, cartKey)
	return this.parseDtoCart(cart)
}

// 创建一个新的购物车
func (this *shoppingService) CreateShoppingCart(merchantId int, memberId int) *dto.ShoppingCart {
	cart := this._rep.GetShopping(merchantId).NewCart(memberId)
	return this.parseDtoCart(cart)
}

func (this *shoppingService) parseDtoCart(c shopping.ICart) *dto.ShoppingCart {
	var cart = new(dto.ShoppingCart)
	v := c.GetValue()
	cart.Id = c.GetDomainId()
	cart.BuyerId = v.BuyerId
	cart.CartKey = v.CartKey
	cart.UpdateTime = v.UpdateTime
	t, f := c.GetFee()
	cart.TotalFee = t
	cart.OrderFee = f
	cart.Summary = c.GetSummary()

	if v.Items != nil {
		if l := len(v.Items); l != 0 {
			cart.Items = make([]*dto.CartItem, l)
			for i, v := range v.Items {
				cart.Items[i] = &dto.CartItem{
					GoodsId:    v.GoodsId,
					GoodsName:  v.Name,
					GoodsNo:    v.GoodsNo,
					SmallTitle: v.SmallTitle,
					GoodsImage: v.Image,
					Num:        v.Quantity,
					Price:      v.Price,
					SalePrice:  v.SalePrice,
				}
				cart.TotalNum += cart.Items[i].Num
			}
		}
	}

	return cart
}

func (this *shoppingService) AddCartItem(merchantId, memberId int, cartKey string, goodsId, num int) (*dto.CartItem, error) {
	cart := this.getShoppingCart(merchantId, memberId, cartKey)
	item, err := cart.AddItem(goodsId, num)
	if err == nil {
		cart.Save()
		return &dto.CartItem{
			GoodsId:    item.GoodsId,
			GoodsName:  item.Name,
			SmallTitle: item.SmallTitle,
			GoodsImage: item.Image,
			Num:        num,
			Price:      item.Price,
			SalePrice:  item.SalePrice,
		}, nil
	}
	return nil, err
}
func (this *shoppingService) SubCartItem(merchantId, memberId int, cartKey string, goodsId, num int) error {
	cart := this.getShoppingCart(merchantId, memberId, cartKey)
	err := cart.RemoveItem(goodsId, num)
	if err == nil {
		_, err = cart.Save()
	}
	return err
}

// 更新购物车结算
func (this *shoppingService) PrepareSettlePersist(merchantId, memberId, shopId, paymentOpt, deliverOpt, deliverId int) error {
	var cart = this.getShoppingCart(merchantId, memberId, "")
	err := cart.SettlePersist(shopId, paymentOpt, deliverOpt, deliverId)
	if err == nil {
		_, err = cart.Save()
	}
	return err
}

func (this *shoppingService) GetCartSettle(merchantId, memberId int, cartKey string) *dto.SettleMeta {
	var cart = this.getShoppingCart(merchantId, memberId, cartKey)
	shop, deliver, payOpt, dlvOpt := cart.GetSettleData()
	var st *dto.SettleMeta = new(dto.SettleMeta)
	st.PaymentOpt = payOpt
	st.DeliverOpt = dlvOpt
	if shop != nil {
		v := shop.GetValue()

		st.Shop = &dto.SettleShopMeta{
			Id:   v.Id,
			Name: v.Name,
			Tel:  v.Phone,
		}
	}

	if deliver != nil {
		v := deliver.GetValue()
		st.Deliver = &dto.SettleDeliverMeta{
			Id:         v.Id,
			PersonName: v.RealName,
			Phone:      v.Phone,
			Address:    v.Address,
		}
	}

	return st
}

func (this *shoppingService) OrderAutoSetup(merchantId int, f func(error)) {
	sp := this._rep.GetShopping(merchantId)
	sp.OrderAutoSetup(f)
}

// 使用余额为订单付款
func (this *shoppingService) PayForOrderWithBalance(merchantId int, orderNo string) error {
	var sp shopping.IShopping = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err == nil {
		err = order.PaymentWithBalance()
	}
	return err
}

// 人工付款
func (this *shoppingService) PayForOrderByManager(merchantId int, orderNo string) error {
	var sp shopping.IShopping = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err == nil {
		err = order.CmPaymentWithBalance()
	}
	return err
}

// 确认付款
func (this *shoppingService) PayForOrderOnlineTrade(merchantId int, orderNo string, spName string, tradeNo string) error {
	var sp shopping.IShopping = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err == nil {
		err = order.PaymentForOnlineTrade(spName, tradeNo)
	}
	return err
}

// 确定订单
func (this *shoppingService) ConfirmOrder(merchantId int, orderNo string) error {
	var sp = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err == nil {
		err = order.Confirm()
	}
	return err
}

// 配送订单,并记录配送服务商编号及单号
func (this *shoppingService) DeliveryOrder(merchantId int, orderNo string,
	deliverySpId int, deliverySpNo string) error {
	//todo:配送订单,并记录配送服务商编号及单号
	var sp = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err == nil && order.GetValue().Status == enum.ORDER_WAIT_DELIVERY {
		err = order.Deliver(deliverySpId, deliverySpNo)
	}
	return err
}

// 标记订单已经收货
func (this *shoppingService) SignOrderReceived(merchantId int, orderNo string) error {
	var sp = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err == nil && order.GetValue().Status == enum.ORDER_WAIT_RECEIVE {
		err = order.SignReceived()
	}
	return err
}

// 标记订单已经完成
func (this *shoppingService) SignOrderCompleted(merchantId int, orderNo string) error {
	var sp = this._rep.GetShopping(merchantId)
	order, err := sp.GetOrderByNo(orderNo)
	if err == nil && order.GetValue().Status == enum.ORDER_RECEIVED {
		err = order.Complete()
	}
	return err
}
