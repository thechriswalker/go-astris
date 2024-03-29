package elgamal

import (
	"strings"

	big "github.com/ncw/gmp"
)

// These are known EG parameters.
// we only expand once, so the convenience is worth it.
func intFrom(hex string) *big.Int {
	n, ok := new(big.Int).SetString(
		strings.Join(strings.Fields(hex), ""),
		16,
	)
	if !ok {
		panic("bad hex decode of EG parameters")
	}
	return n
}

// Mini one for testing
func EightBit() *System {
	return &System{P: big.NewInt(227), Q: big.NewInt(113), G: big.NewInt(69)}
}

// DH1024modp160 elGamal params from RFC 5114
func DH1024modp160() *System {
	return &System{
		P: intFrom(`
			B10B8F96 A080E01D DE92DE5E AE5D54EC 52C99FBC FB06A3C6
			9A6A9DCA 52D23B61 6073E286 75A23D18 9838EF1E 2EE652C0
			13ECB4AE A9061123 24975C3C D49B83BF ACCBDD7D 90C4BD70
			98488E9C 219A7372 4EFFD6FA E5644738 FAA31A4F F55BCCC0
			A151AF5F 0DC8B4BD 45BF37DF 365C1A65 E68CFDA7 6D4DA708
			DF1FB2BC 2E4A4371`),
		G: intFrom(`
			A4D1CBD5 C3FD3412 6765A442 EFB99905 F8104DD2 58AC507F
			D6406CFF 14266D31 266FEA1E 5C41564B 777E690F 5504F213
			160217B4 B01B886A 5E91547F 9E2749F4 D7FBD7D3 B9A92EE1
			909D0D22 63F80A76 A6A24C08 7A091F53 1DBF0A01 69B6A28A
			D662A4D1 8E73AFA3 2D779D59 18D08BC8 858F4DCE F97C2A24
			855E6EEB 22B3B2E5`),
		Q: intFrom(`
			F518AA87 81A8DF27 8ABA4E7D 64B7CB9D 49462353`),
	}
}

// DH2048modp224 elGamal params from RFC 5114
func DH2048modp224() *System {
	return &System{
		P: intFrom(`
			AD107E1E 9123A9D0 D660FAA7 9559C51F A20D64E5 683B9FD1
			B54B1597 B61D0A75 E6FA141D F95A56DB AF9A3C40 7BA1DF15
			EB3D688A 309C180E 1DE6B85A 1274A0A6 6D3F8152 AD6AC212
			9037C9ED EFDA4DF8 D91E8FEF 55B7394B 7AD5B7D0 B6C12207
			C9F98D11 ED34DBF6 C6BA0B2C 8BBC27BE 6A00E0A0 B9C49708
			B3BF8A31 70918836 81286130 BC8985DB 1602E714 415D9330
			278273C7 DE31EFDC 7310F712 1FD5A074 15987D9A DC0A486D
			CDF93ACC 44328387 315D75E1 98C641A4 80CD86A1 B9E587E8
			BE60E69C C928B2B9 C52172E4 13042E9B 23F10B0E 16E79763
			C9B53DCF 4BA80A29 E3FB73C1 6B8E75B9 7EF363E2 FFA31F71
			CF9DE538 4E71B81C 0AC4DFFE 0C10E64F`),
		G: intFrom(`
			AC4032EF 4F2D9AE3 9DF30B5C 8FFDAC50 6CDEBE7B 89998CAF
			74866A08 CFE4FFE3 A6824A4E 10B9A6F0 DD921F01 A70C4AFA
			AB739D77 00C29F52 C57DB17C 620A8652 BE5E9001 A8D66AD7
			C1766910 1999024A F4D02727 5AC1348B B8A762D0 521BC98A
			E2471504 22EA1ED4 09939D54 DA7460CD B5F6C6B2 50717CBE
			F180EB34 118E98D1 19529A45 D6F83456 6E3025E3 16A330EF
			BB77A86F 0C1AB15B 051AE3D4 28C8F8AC B70A8137 150B8EEB
			10E183ED D19963DD D9E263E4 770589EF 6AA21E7F 5F2FF381
			B539CCE3 409D13CD 566AFBB4 8D6C0191 81E1BCFE 94B30269
			EDFE72FE 9B6AA4BD 7B5A0F1C 71CFFF4C 19C418E1 F6EC0179
			81BC087F 2A7065B3 84B890D3 191F2BFA`),
		Q: intFrom(`
			801C0D34 C58D93FE 99717710 1F80535A 4738CEBC BF389A99
			B36371EB`),
	}
}

// DH2048modp256 elGamal params from RFC 5114
func DH2048modp256() *System {
	return &System{
		P: intFrom(`
			87A8E61D B4B6663C FFBBD19C 65195999 8CEEF608 660DD0F2
			5D2CEED4 435E3B00 E00DF8F1 D61957D4 FAF7DF45 61B2AA30
			16C3D911 34096FAA 3BF4296D 830E9A7C 209E0C64 97517ABD
			5A8A9D30 6BCF67ED 91F9E672 5B4758C0 22E0B1EF 4275BF7B
			6C5BFC11 D45F9088 B941F54E B1E59BB8 BC39A0BF 12307F5C
			4FDB70C5 81B23F76 B63ACAE1 CAA6B790 2D525267 35488A0E
			F13C6D9A 51BFA4AB 3AD83477 96524D8E F6A167B5 A41825D9
			67E144E5 14056425 1CCACB83 E6B486F6 B3CA3F79 71506026
			C0B857F6 89962856 DED4010A BD0BE621 C3A3960A 54E710C3
			75F26375 D7014103 A4B54330 C198AF12 6116D227 6E11715F
			693877FA D7EF09CA DB094AE9 1E1A1597`),
		G: intFrom(`
			3FB32C9B 73134D0B 2E775066 60EDBD48 4CA7B18F 21EF2054
			07F4793A 1A0BA125 10DBC150 77BE463F FF4FED4A AC0BB555
			BE3A6C1B 0C6B47B1 BC3773BF 7E8C6F62 901228F8 C28CBB18
			A55AE313 41000A65 0196F931 C77A57F2 DDF463E5 E9EC144B
			777DE62A AAB8A862 8AC376D2 82D6ED38 64E67982 428EBC83
			1D14348F 6F2F9193 B5045AF2 767164E1 DFC967C1 FB3F2E55
			A4BD1BFF E83B9C80 D052B985 D182EA0A DB2A3B73 13D3FE14
			C8484B1E 052588B9 B7D2BBD2 DF016199 ECD06E15 57CD0915
			B3353BBB 64E0EC37 7FD02837 0DF92B52 C7891428 CDC67EB6
			184B523D 1DB246C3 2F630784 90F00EF8 D647D148 D4795451
			5E2327CF EF98C582 664B4C0F 6CC41659`),
		Q: intFrom(`
			8CF83642 A709A097 B4479976 40129DA2 99B1A47D 1EB3750B
			A308B0FE 64F5FBD3`),
	}
}

// Helios ElGamal params from the source code:
// it is a 2048modp256 system
// I do not know the provenance of the numbers.
func Helios() *System {
	return &System{
		P: intFrom(`
			815901AA 6D3CED6A 0BD488C6 17351E32 2C8AEF8F 9F90ADF3
			31A2583D 8082AC46 F74345A1 E1CF561F ACBDF323 9BC3F0EE
			71618B5D 016266CA AFD48439 B034A38F 6560CD6B 671E3A80
			248B4680 9AD8DE7A 4CC72504 69611D59 DAE8D8AF 5C6D0F9F
			3665F985 7E04E113 4DC94B27 0E933414 49EA5036 17447ECB
			83B2C016 02878C07 0D080DA4 64C974D9 951C35C1 A5534073
			45EE31EB C4A29A34 88D5A547 02A971EE 0A1EA4DA 93FCF641
			05040893 FF4BEC23 CA11E8CF FA279E89 9A468911 37C28E85
			F5A2FC9C 637AF6D2 6F6B5DEB A3D60580 DF41C334 EA123331
			F8B0ADEB 43EA64A0 37E0C5AC 168C47CE 421BC971 8BA83570
			99A0221F 778599AC D917607F 3E3024D7`),
		G: intFrom(`
			75EE80F0 A161DD0C 025AC818 DB8D52D1 93A46655 FE0EBD3C
			289A949F 42185F58 F2F88F82 5DCDB3E3 E98C0598 AF875997
			28F4F071 9A8F68B1 33E82EB1 BC4E3B6B 8A377A5C 6B812D65
			6EFCDE57 8FDF515A C6EF628F 1564AC90 7745D53B C6213B74
			F0CC303B BE68F3AB 2220DCAC D0CEECE7 AAC3A675 AAA06048
			85A1FB13 74E6C08F 2DCF503E 58AC6487 BE73B8AB 2A10FA62
			A79522CB C777B632 1FD346E0 D36EE5A7 29195511 7D8BB428
			4901EB26 804BD228 6A14AF52 F5301C48 9C80DFEA FB7CE496
			AF58479A 4C6F57F2 9EC8C9E4 F6B88DEB 06F5D120 859D2D4D
			E06E57B0 476F8263 F7A4A35F 67ED21A4 A927109F A89A6B7F
			4976E98E 3DDB3CD2 32C516B1 DA5CC555`),
		Q: intFrom(`
			87974DEB 793421CE 3891540D 906AC080 6B85A2B9 5ADC211A
			82EF8B65 9F8D9D25`),
	}
}

// This is a 2048bit prime with a 2047bit subgroup and P = 2Q +1
// Safe for use, but Q is large compared to the other systems.
func Astris2048() *System {
	return &System{
		P: intFrom(`
			FF1C5DB8 C1323A2F 0E254EAB FDA41B3F EA6EC253 2E348A88
			317E59AD F627B356 E9338D74 5AA3B9A5 9F3DCA50 B16E00DE
			4CEFEDC2 B7F21AB2 982F6E35 2539E956 38940FD0 00A7C9D3
			EF542A17 8AC5E79E 2C0AE4A9 30FFCCEC A9B8CB5F 342BEE00
			939EEF7C 6E90C88F 39E5C7EA 11CAF8AF 65F8BC1A A484B369
			596DAE5E 8F25DAA7 833AE4AE BD116050 708DAAA7 5106E22F
			922C6D6D A9143D29 4AC10E59 B49327E3 21F6C91E C2EF1879
			E5A7663B AC19618E 8BBE3825 B24848DE 0F8B1B2C 3A49C106
			B25FF1FB D47F2E0C 6688EB97 85443FD9 06E48D32 46F8D9F4
			35AEA9BD EE2735FB 61867607 3E37AD54 79E335BA E8C326E6
			AF2B9C17 C3E75478 81756743 E9BD3937`),
		G: intFrom(`
			F41B9E98 FD6A505A 93289A49 2D68F388 3E67D34C ACB7F674
			92AED386 28EA3AC8 47C1EE64 6ED05926 052CD88D 3FA11230
			92C726A3 579BC2FC 631A2BC9 5A0FFBFF 4338655E BDCB6D03
			DAF42E56 81E771C8 BA1C9DB7 09968194 DC82D5C3 BE006DC6
			1CEEAF50 777533C6 7CE1E303 ADF500E7 1AC86435 921C2B17
			54A29545 DA1668EE 2005CD14 90502ACB 9187B6D0 CFC70625
			7602BB7A 1DB125AC 86518C5F 04A83587 4A4FC7C7 49F0B76D
			078E18AF C38448A1 40CEDE09 2EA15486 BD41A5C4 9CFE92A5
			364F4E01 F197BF36 2B462237 2B9A3A00 A0DB0A52 4F71448A
			1AD0B2AC EEBDD1F3 26A258F6 45FEC21B 782C12E3 37128576
			9A08A28F 6AB34ABC 1E6092E5 439DF376`),
		Q: intFrom(`
			7F8E2EDC 60991D17 8712A755 FED20D9F F5376129 971A4544
			18BF2CD6 FB13D9AB 7499C6BA 2D51DCD2 CF9EE528 58B7006F
			2677F6E1 5BF90D59 4C17B71A 929CF4AB 1C4A07E8 0053E4E9
			F7AA150B C562F3CF 16057254 987FE676 54DC65AF 9A15F700
			49CF77BE 37486447 9CF2E3F5 08E57C57 B2FC5E0D 524259B4
			ACB6D72F 4792ED53 C19D7257 5E88B028 3846D553 A8837117
			C91636B6 D48A1E94 A560872C DA4993F1 90FB648F 61778C3C
			F2D3B31D D60CB0C7 45DF1C12 D924246F 07C58D96 1D24E083
			592FF8FD EA3F9706 334475CB C2A21FEC 83724699 237C6CFA
			1AD754DE F7139AFD B0C33B03 9F1BD6AA 3CF19ADD 74619373
			5795CE0B E1F3AA3C 40BAB3A1 F4DE9C9B`),
	}
}
